.PHONY: build build-cli build-server run dev test clean docker-up docker-down

# Build all
build: build-cli build-server

# Build CLI
build-cli:
	go build -o task ./cmd/irontask

# Build server
build-server:
	go build -o irontask-server ./cmd/irontask-server

# Run CLI (TUI mode)
run: build-cli
	./task

# Run server locally
dev: build-server
	DATABASE_URL="postgres://irontask:irontask@localhost:5432/irontask?sslmode=disable" ./irontask-server

# Run Web UI
run-ui:
	cd web-ui && npm run dev

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f task irontask-server

# Docker compose up
docker-up:
	docker-compose up -d

# Docker compose down
docker-down:
	docker-compose down

# Docker compose logs
docker-logs:
	docker-compose logs -f

# Start only postgres
postgres:
	docker-compose up -d postgres

# Tidy modules
tidy:
	go mod tidy

# Install (copy to /usr/local/bin)
install: build-cli
	cp task /usr/local/bin/task

# Help
help:
	@echo "IronTask Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build         Build both CLI and server"
	@echo "  make build-cli     Build CLI only"
	@echo "  make build-server  Build server only"
	@echo "  make run           Run TUI"
	@echo "  make dev           Run server locally"
	@echo "  make test          Run tests"
	@echo "  make clean         Clean build artifacts"
	@echo "  make docker-up     Start docker containers"
	@echo "  make docker-down   Stop docker containers"
	@echo "  make postgres      Start only PostgreSQL"
	@echo "  make install       Install CLI to /usr/local/bin"
