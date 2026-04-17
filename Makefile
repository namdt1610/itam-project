# Default target
all: build-agent build-server

# ===== AGENT BUILDS =====

# Build Windows agent (dev - requires config.json)
build-agent:
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o ./build/agent.exe ./agent/main.go

# Build Windows agent for PRODUCTION (single exe with embedded config)
# Usage: make build-agent-prod SERVER_URL=http://192.168.1.100:8080/api/report AUTH_TOKEN=your-token
SERVER_URL ?= http://localhost:8080/api/report
AUTH_TOKEN ?= 
build-agent-prod:
	GOOS=windows GOARCH=amd64 go build \
		-ldflags "-s -w -X main.BuildServerURL=$(SERVER_URL) -X main.BuildAuthToken=$(AUTH_TOKEN)" \
		-o ./build/agent.exe ./agent/main.go
	@echo "Built agent.exe with embedded config: $(SERVER_URL)"

# Build Linux agent executable
build-linux-agent:
	go build -ldflags "-s -w" -o ./build/agent_bin ./agent/main.go

# ===== SERVER BUILDS =====

# Build Linux server executable
build-server:
	@mkdir -p data
	go build -ldflags "-s -w" -o ./build/server_bin ./server

# ===== RUN COMMANDS =====

# Run server for development
run-server:
	@mkdir -p data
	go run ./server

# Run agent for development (Linux)
run-agent:
	go run ./agent/main.go

# Simulate GPO: Run Linux agent every 5 seconds
run-agent-loop: build-linux-agent
	@echo "Starting agent loop (CTRL+C to stop)..."
	@while true; do ./build/agent_bin; sleep 5; done

# Run load test (500 simulated agents)
loadtest:
	go run ./cmd/loadtest

# ===== UTILITIES =====

# Clean build artifacts
clean:
	rm -rf ./build/*

# Install dependencies
deps:
	go mod download
	go mod tidy

.PHONY: all build-agent build-agent-prod build-linux-agent build-server run-server run-agent run-agent-loop loadtest clean deps

