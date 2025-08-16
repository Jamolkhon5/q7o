.PHONY: help run build test clean docker-up docker-down livekit-up livekit-down logs livekit-logs network-create

help:
	@echo "Available commands:"
	@echo "  make run              - Run the application locally"
	@echo "  make build            - Build the application"
	@echo "  make test             - Run tests"
	@echo "  make network-create   - Create Docker network"
	@echo "  make docker-up        - Start app, DB, Redis"
	@echo "  make docker-down      - Stop app containers"
	@echo "  make livekit-up       - Start LiveKit server"
	@echo "  make livekit-down     - Stop LiveKit server"
	@echo "  make logs             - Show app logs"
	@echo "  make livekit-logs     - Show LiveKit logs"
	@echo "  make clean            - Clean everything"

run:
	go run cmd/api/main.go

build:
	go build -o bin/q7o cmd/api/main.go

test:
	go test -v ./...

network-create:
	-docker network create q7o_network

docker-up:
	-docker network create q7o_network
	docker-compose up -d --build

docker-down:
	docker-compose down

livekit-up:
	-docker network create q7o_network
	docker-compose -f docker-compose.livekit.yml up -d

livekit-down:
	docker-compose -f docker-compose.livekit.yml down

logs:
	docker-compose logs -f app

livekit-logs:
	docker-compose -f docker-compose.livekit.yml logs -f

clean:
	docker-compose down -v
	docker-compose -f docker-compose.livekit.yml down -v
	-docker network rm q7o_network
	rm -rf bin/

# Shortcuts
up: docker-up livekit-up
down: docker-down livekit-down
restart: down up