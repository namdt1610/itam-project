# Default target
all: build-agent build-server

# Build Windows agent executable
build-agent:
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o ./build/agent.exe ./agent/main.go

# Build Linux agent executable
build-linux-agent:
	go build -ldflags "-s -w" -o ./build/agent_bin ./agent/main.go

# Build Linux server executable
build-server:
	go build -ldflags "-s -w" -o server_bin ./server

# Run server for development
run-server:
	go run ./server

# Run agent for development (Linux)
run-agent:
	go run ./agent/main.go

# Simulate GPO: Run Linux agent every 5 seconds
run-agent-loop: build-linux-agent
	@echo "Starting agent loop (CTRL+C to stop)..."
	@while true; do ./build/agent_bin; sleep 5; done

# Clean build artifacts
clean:
	rm -f agent.exe server_bin

# Install dependencies
deps:
	go mod download
	go mod tidy

.PHONY: all build-agent build-server run-server run-agent clean deps
