FRONTEND_DIR = ./web
BACKEND_DIR = .
IMAGE := altronsoft/new-api
TAG := latest
PLATFORM ?= linux/amd64

.PHONY: all build-frontend start-backend build push

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

build:
	@echo "Building Docker image with embedded frontend..."
	docker buildx build --platform $(PLATFORM) -t $(IMAGE):$(TAG) -f Dockerfile.server .

push:
	docker push $(IMAGE):$(TAG)

pull:
	docker pull $(IMAGE):$(TAG)

up:
	docker compose up -d

log:
	docker compose logs -f new-api