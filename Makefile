BINARY     := vidian
CMD        := ./cmd/vidian/main.go
INSTALL_DIR := /usr/local/bin
FRONTEND   := ./frontend

.PHONY: all build install uninstall dev clean help

## help: Show this help message
help:
	@echo ""
	@echo "  Vidian — Build & Install Commands"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/^## //' | awk -F': ' '{printf "  \033[36mmake %-14s\033[0m %s\n", $$1, $$2}'
	@echo ""

## all: Build frontend and compile binary
all: frontend build

## frontend: Install npm deps and build Svelte assets
frontend:
	@echo "→ Building frontend assets..."
	cd $(FRONTEND) && npm install && npm run build
	@echo "✓ Frontend built."

## build: Compile Go binary (requires frontend to be built first)
build:
	@echo "→ Compiling Go binary..."
	go build -ldflags="-s -w" -o $(BINARY) $(CMD)
	@echo "✓ Binary ready: ./$(BINARY)"

## install: Build everything and install to $(INSTALL_DIR)
install: all
	@echo "→ Installing $(BINARY) to $(INSTALL_DIR)..."
	@if [ ! -w "$(INSTALL_DIR)" ]; then \
		sudo cp $(BINARY) $(INSTALL_DIR)/$(BINARY); \
	else \
		cp $(BINARY) $(INSTALL_DIR)/$(BINARY); \
	fi
	@echo "✓ Installed! Run: $(BINARY) ."

## uninstall: Remove installed binary from $(INSTALL_DIR)
uninstall:
	@echo "→ Removing $(BINARY) from $(INSTALL_DIR)..."
	@if [ ! -w "$(INSTALL_DIR)" ]; then \
		sudo rm -f $(INSTALL_DIR)/$(BINARY); \
	else \
		rm -f $(INSTALL_DIR)/$(BINARY); \
	fi
	@echo "✓ Uninstalled."

## dev: Show dev mode instructions
dev:
	@echo ""
	@echo "  Run these in two separate terminals:"
	@echo ""
	@echo "  Terminal 1 — Svelte dev server with HMR:"
	@echo "    cd frontend && npm run dev"
	@echo ""
	@echo "  Terminal 2 — Go backend in dev mode:"
	@echo "    go run $(CMD) -dir . -dev -port 8080"
	@echo ""

## clean: Remove build artifacts
clean:
	@rm -f $(BINARY)
	@rm -rf $(FRONTEND)/dist
	@echo "✓ Cleaned."
