package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"lsp-path-translator/proxy"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return fmt.Sprintf("%v", *i)
}
func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	logFile := flag.String("log-file", "", "Save logging to file")

	logMessages := flag.Bool("log-messages", false, "Log translated LSP messages")

	var pathMap arrayFlags
	flag.Var(&pathMap, "path-map", "Map from client to server in format client::server")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] -- <lsp-command> [args...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Fatalf("error opening log file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	args := flag.Args()
	if len(args) == 0 {
		log.Println("Error: No LSP command specified")
		flag.Usage()
		os.Exit(1)
	}

	lspCommand := args[0]
	lspArgs := args[1:]

	// Create translators
	pathMapArr := []string(pathMap)

	clientToServer, err := proxy.NewJSONPathTranslators(pathMapArr)
	if err != nil {
		log.Fatalf("Failed to start LSP command: %v", err)
	}

	serverToClient, err := proxy.NewJSONPathTranslators(pathMapArr)
	if err != nil {
		log.Fatalf("Failed to start LSP command: %v", err)
	}
	for i := range serverToClient {
		serverToClient[i].Invert()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, lspCommand, lspArgs...)

	// Server Stdin/Stdout pipes
	serverStdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to create stdin pipe: %v", err)
	}
	serverStdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}

	// Pass through Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start LSP command: %v", err)
	}

	log.Printf("Started proxy for %s mapping %q\n", lspCommand, pathMapArr)

	// Goroutine 1: Client to Server (Stdin -> ServerStdin)
	go func() {
		defer serverStdin.Close()
		stream := proxy.NewStreamRW(os.Stdin, serverStdin, &clientToServer, *logMessages)
		for {
			payload, err := stream.ReadAndTranslate()
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					log.Println("Client closed connection")
				} else {
					log.Printf("Error reading from client: %v", err)
				}
				cancel()
				return
			}
			if err := stream.Write(payload); err != nil {
				log.Printf("Error writing to server: %v", err)
				cancel()
				return
			}
		}
	}()

	// Goroutine 2: Server to Client (ServerStdout -> Stdout)
	go func() {
		stream := proxy.NewStreamRW(serverStdout, os.Stdout, &serverToClient, *logMessages)
		for {
			payload, err := stream.ReadAndTranslate()
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF || err == os.ErrClosed {
					log.Println("Server closed connection")
				} else {
					log.Printf("Error reading from server: %v", err)
				}
				cancel()
				return
			}
			if err := stream.Write(payload); err != nil {
				log.Printf("Error writing to client: %v", err)
				cancel()
				return
			}
		}
	}()

	// Handle signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigs
		log.Println("Received termination signal, stopping...")
		cancel()
	}()

	// Wait for process to exit
	err = cmd.Wait()
	log.Printf("LSP process exited: %v\n", err)

	// Ensure everything shuts down
	cancel()
}
