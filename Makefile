BINARY_NAME=boilerplate-go
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
.DEFAULT_GOAL := run

install:
	go mod download
	go mod verify
	go mod tidy

clean:
	go clean
	rm -rf bin

lint:
	golangci-lint run -v

lint_fix:
	golangci-lint run -v --fix

.PHONY: docs
docs:
	swag init -q -g cmd/api/main.go -o cmd/api/docs

build: lint_fix build_api

build_run: lint_fix build_run_api

run: lint_fix run_api

run_api:
	cd cmd/api && go run main.go || cd -

run_pub:
	cd cmd/test/rabbit/pub && go run main.go || cd -

run_sub:
	cd cmd/test/rabbit/sub && go run main.go || cd -

run_rpc_sender:
	cd cmd/test/rabbit/rpc_sender && go run main.go || cd -

run_rpc_receiver:
	cd cmd/test/rabbit/rpc_receiver && go run main.go || cd -

build_api:
	GOARCH=amd64 GOOS=darwin go build -o bin/api/api-$(BINARY_NAME)-darwin ./cmd/api/main.go
	GOARCH=amd64 GOOS=linux go build -o bin/api/api-$(BINARY_NAME)-linux ./cmd/api/main.go
	cp ./cmd/api/.env ./bin/api/.env

build_run_api: build_api
	cd bin/api && ./api-$(BINARY_NAME)-darwin || cd -

podman_build_api:
	podman build -t api-$(BINARY_NAME):latest -f cmd/api/.Dockerfile .

podman_run_api:
	podman run -d -p 8001:8001 \
      -v $(ROOT_DIR)/cmd/api/.env:/app/.env \
      --name api-$(BINARY_NAME) localhost/api-$(BINARY_NAME):latest

podman_stop_api:
	podman stop api-$(BINARY_NAME)
	podman rm -f api-$(BINARY_NAME)

podman_remove_api:
	podman stop api-$(BINARY_NAME)
	podman rmi -f api-$(BINARY_NAME):latest
