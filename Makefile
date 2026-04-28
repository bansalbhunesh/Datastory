.PHONY: dev mock mock-down backend frontend install test test-backend test-frontend lint build

# Run OpenMetadata quickstart (official compose bundled under docker/openmetadata).
mock:
	cd docker/openmetadata && docker compose up -d

mock-down:
	cd docker/openmetadata && docker compose down

install:
	cd backend && go mod tidy
	cd frontend && npm install

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

# Parallel backend + frontend (requires GNU make).
dev:
	$(MAKE) -j2 backend frontend

# Run every test suite. Use this in CI before tagging a release.
test: test-backend test-frontend

test-backend:
	cd backend && go test ./... -race -count=1

test-frontend:
	cd frontend && npm test

# Static analysis. golangci-lint isn't required; go vet covers the basics.
lint:
	cd backend && go vet ./...

# Production build: compiles the React app, copies it into backend/dist, and
# builds the Go server binary that serves both the API and the SPA.
build:
	cd frontend && npm ci && npm run build
	rm -rf backend/dist
	cp -r frontend/dist backend/dist
	cd backend && go build -o bin/datastory ./cmd/server
