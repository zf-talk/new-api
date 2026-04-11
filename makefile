FRONTEND_DIR = ./web
BACKEND_DIR = .
IMAGE := altronsoft/new-api
TAG := latest

.PHONY: all build-frontend start-backend build push

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

build:
	@echo "Building frontend locally..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(shell cat VERSION) bun run build
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE):$(TAG) -f Dockerfile.server .

push:
	docker push $(IMAGE):$(TAG)