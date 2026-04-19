.PHONY: dev mock mock-down backend frontend install

# Run OpenMetadata quickstart (official compose bundled under docker/openmetadata).
mock:
	cd docker/openmetadata && docker compose up -d

mock-down:
	cd docker/openmetadata && docker compose down

install:
	cd backend && go mod tidy
	cd frontend && npm install

backend:
	cd backend && go run .

frontend:
	cd frontend && npm run dev

# Parallel backend + frontend (requires GNU make).
dev:
	$(MAKE) -j2 backend frontend
