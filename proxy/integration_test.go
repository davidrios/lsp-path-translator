package proxy

import (
	"encoding/json"
	"io"
	"sync"
	"testing"
)

func TestProxyIntegration_TwoWay(t *testing.T) {
	// Setup pipes.
	// io.Pipe returns (Reader, Writer)

	// Client -> Proxy -> Server
	proxyIn, clientOut := io.Pipe() // Client writes to clientOut, proxy reads from proxyIn
	serverIn, proxyOut := io.Pipe() // Proxy writes to proxyOut, server reads from serverIn

	// Server -> Proxy -> Client
	proxyBackIn, serverOut := io.Pipe() // Server writes to serverOut, proxy reads from proxyBackIn
	clientIn, proxyBackOut := io.Pipe() // Proxy writes to proxyBackOut, client reads from clientIn

	clientToServer := NewJSONPathTranslator("/client/code", "/server/code")
	serverToClient := NewJSONPathTranslator("/server/code", "/client/code")

	var wg sync.WaitGroup
	wg.Add(2)

	// Proxy loop: Client to Server
	go func() {
		defer wg.Done()
		stream := NewStreamRW(proxyIn, proxyOut, clientToServer)
		for {
			payload, err := stream.ReadAndTranslate()
			if err != nil {
				proxyOut.Close()
				return
			}
			err = stream.Write(payload)
			if err != nil {
				proxyOut.Close()
				return
			}
		}
	}()

	// Proxy loop: Server to Client
	go func() {
		defer wg.Done()
		stream := NewStreamRW(proxyBackIn, proxyBackOut, serverToClient)
		for {
			payload, err := stream.ReadAndTranslate()
			if err != nil {
				proxyBackOut.Close()
				return
			}
			err = stream.Write(payload)
			if err != nil {
				proxyBackOut.Close()
				return
			}
		}
	}()

	// Test orchestrator wrappers
	clientStream := NewStreamRW(clientIn, clientOut, NewJSONPathTranslator("", ""))
	serverStream := NewStreamRW(serverIn, serverOut, NewJSONPathTranslator("", ""))

	// 1. Client sends request to Server
	clientReq := []byte(`{"method":"initialize","rootUri":"file:///client/code"}`)
	go func() {
		err := clientStream.Write(clientReq)
		if err != nil {
			t.Errorf("Client failed to write: %v", err)
		}
	}()

	// Server receives request
	serverRecv, err := serverStream.ReadAndTranslate()
	if err != nil {
		t.Fatalf("Server failed to read: %v", err)
	}

	// Verify server got translated request
	var srvJSON map[string]any
	err = json.Unmarshal(serverRecv, &srvJSON)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if srvJSON["rootUri"] != "file:///server/code" {
		t.Errorf("Expected server to receive file:///server/code, got %v", srvJSON["rootUri"])
	}

	// 2. Server sends response to Client
	serverRes := []byte(`{"result":{"path":"/server/code/file.go"}}`)
	go func() {
		err := serverStream.Write(serverRes)
		if err != nil {
			t.Errorf("Server failed to write: %v", err)
		}
	}()

	// Client receives response
	clientRecv, err := clientStream.ReadAndTranslate()
	if err != nil {
		t.Fatalf("Client failed to read: %v", err)
	}

	// Verify client got translated response
	var clJSON map[string]any
	err = json.Unmarshal(clientRecv, &clJSON)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resObj := clJSON["result"].(map[string]any)
	if resObj["path"] != "/client/code/file.go" {
		t.Errorf("Expected client to receive /client/code/file.go, got %v", resObj["path"])
	}

	// Cleanup
	clientOut.Close()
	serverOut.Close()
	proxyIn.Close()
	proxyBackIn.Close()

	wg.Wait()
}
