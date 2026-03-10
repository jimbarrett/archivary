.PHONY: build build-frontend build-backend run dev dev-frontend dev-backend clean

# Build everything
build: build-frontend build-backend

# Build the Vue frontend
build-frontend:
	cd frontend && npm install --silent && npm run build

# Build the Go binary
build-backend: build-frontend
	go build -ldflags "-X main.version=dev" -o bin/archivary ./cmd/archivary

# Run the built binary
run: build
	./bin/archivary serve

# Development: run backend and frontend with hot-reload
dev:
	@echo "Starting backend and frontend in dev mode..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:5173 (proxies API to backend)"
	@echo ""
	@echo "Run these in separate terminals:"
	@echo "  make dev-backend"
	@echo "  make dev-frontend"

# Run backend in dev mode
dev-backend:
	go run ./cmd/archivary serve

# Run frontend dev server with hot-reload
dev-frontend:
	cd frontend && npm run dev

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf frontend/dist/
