# LSP Proxy

LSP Proxy is an intermediate proxy for the Language Server Protocol (LSP). It allows a client to communicate with an LSP server by transparently translating file paths in the requests and responses between the two.

This is extremely useful when the LSP server runs in a different environment than the client (e.g., inside a Docker container, an SSH session, or a mounted volume with a different root path structure).

## Features

- Deep JSON inspection for paths and URIs (modifies strings like `/client/path` to `/server/path` and `file:///client/path` to `file:///server/path`).
- Configurable base paths via command-line flags.
- Directly wraps your desired LSP language server transparently.
- Implemented in Go for extreme performance and zero dependencies.

## Installation

To build the executable proxy:

```bash
git clone <repository>
cd lspproxy
go build -o bin/lspproxy main.go
```

## Usage

You must specify the `-client-path` and `-server-path` flags, which specify the prefix replacement operation to apply.
Trailing arguments after `--` represent the underlying language server command to launch.

### Example: Go (`gopls`)

Let's assume our actual source code exists at `/host/David/projects/app`, but our gopls language server and our Go environment thinks it lives at `/container/app`:

```bash
./bin/lspproxy -client-path /host/David/projects/app -server-path /container/app -- gopls
```

### Example: Python (`pyright`)

```bash
./bin/lspproxy -client-path /mnt/c/Users/David/project -server-path /usr/src/project -- pyright-langserver --stdio
```

## Development and Testing

Developing the proxy requires Go `1.21` or later.

The path modification and proxy streams are fully tested with unit and integration tests. No manual end-to-end integration binary spawns are required to ensure the translation stream passes through the pipes properly.

To run tests:

```bash
go test ./... -v
```

This will run all tests, including:
- `JSONPathTranslator` structural translation tests.
- LSP content-length framing streams.
- Full two-way integration tests using memory-bound `io.Pipe()`.
