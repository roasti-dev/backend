GO ?= go

FIREBASE_TOOLS_VERSION := 15.10.0@sha256:740d133bffbcda740b49f7e5ce883ecf7412752a931c68a6ad2040a0622e03a4
FIREBASE_PROJECT := roasti-dev-project
FIREBASE_AUTH_PORT := 9099
FIREBASE_CONTAINER_NAME := firebase-emulator-dev

OUTPUT_DEBIAN := app-debian

STANDARD_ENUMS := ./internal/recipe/brew_method.go \
	./internal/recipe/difficulty.go

NULLABLE_ENUMS := ./internal/recipe/roast_level.go

STANDARD_ENUMS_GO := $(STANDARD_ENUMS:.go=_enum.go)
NULLABLE_ENUMS_GO := $(NULLABLE_ENUMS:.go=_enum.go)

$(STANDARD_ENUMS_GO): GO_ENUM_FLAGS=--marshal --names --sqlint
$(NULLABLE_ENUMS_GO): GO_ENUM_FLAGS=--marshal --names --sqlnullint

enums: $(STANDARD_ENUMS_GO) $(NULLABLE_ENUMS_GO)

%_enum.go: %.go
	$(GO) tool go-enum -f $< $(GO_ENUM_FLAGS)

OAPI_SPEC := api/spec.yaml
OAPI_CONFIG := api/spec-config.yaml
OAPI_MODELS_CONFIG := api/models-config.yaml
OAPI_MODELS := api/models.yaml
OAPI_OUT := internal/handlers/server.gen.go
OAPI_MODELS_OUT := internal/api/models/models.gen.go
OAPI_CLIENT_CONFIG := api/client-config.yaml
OAPI_CLIENT_OUT := tests/client/client.gen.go

$(OAPI_MODELS_OUT): $(OAPI_MODELS) $(OAPI_MODELS_CONFIG)
	$(GO) tool oapi-codegen -config $(OAPI_MODELS_CONFIG) -o $(OAPI_MODELS_OUT) $(OAPI_MODELS)

$(OAPI_OUT): $(OAPI_SPEC) $(OAPI_CONFIG) $(OAPI_MODELS_OUT)
	$(GO) tool oapi-codegen -config $(OAPI_CONFIG) -o $(OAPI_OUT) $(OAPI_SPEC)

$(OAPI_CLIENT_OUT): $(OAPI_SPEC) $(OAPI_CLIENT_CONFIG) $(OAPI_MODELS_OUT)
	$(GO) tool oapi-codegen -config $(OAPI_CLIENT_CONFIG) -o $(OAPI_CLIENT_OUT) $(OAPI_SPEC)

oapi: $(OAPI_MODELS_OUT) $(OAPI_OUT) $(OAPI_CLIENT_OUT)

build:
	$(GO) build -o app ./cmd/server

# Debian 13 (Trixie) 64-bit
build-debian:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		$(GO) build -ldflags="-s -w" -o $(OUTPUT_DEBIAN) ./cmd/server

start:
	APP_ENV=development DEBUG=true $(GO) run ./cmd/server

DEPLOY_USER ?= root
DEPLOY_PATH ?= /home/app
BACKEND_SERVICE := backend
REMOTE := $(DEPLOY_USER)@$(DEPLOY_HOST)
REMOTE_BIN := $(DEPLOY_PATH)/$(OUTPUT_DEBIAN)

deploy: build-debian
ifndef DEPLOY_HOST
	$(error DEPLOY_HOST is not set)
endif
	scp $(OUTPUT_DEBIAN) $(REMOTE):$(REMOTE_BIN).new
	scp .env.production $(REMOTE):$(REMOTE_DIR)/.env
	ssh $(REMOTE) '\
		mv $(REMOTE_BIN).new $(REMOTE_BIN) && \
		chmod +x $(REMOTE_BIN) && \
		sudo systemctl restart $(BACKEND_SERVICE) && \
		sudo systemctl status $(BACKEND_SERVICE) --no-pager \
	'

lint:
	golangci-lint run

test-e2e: firebase-emulator wait-firebase
	APP_ENV=development \
	FIREBASE_PROJECT_ID=$(FIREBASE_PROJECT) \
	FIREBASE_API_KEY=test \
	FIREBASE_AUTH_EMULATOR_HOST=localhost:$(FIREBASE_AUTH_PORT) \
	FIREBASE_IDENTITY_BASE_URL=http://localhost:$(FIREBASE_AUTH_PORT)/identitytoolkit.googleapis.com/v1/accounts \
	FIREBASE_TOKEN_BASE_URL=http://localhost:$(FIREBASE_AUTH_PORT)/securetoken.googleapis.com/v1/token \
	go test -v -coverprofile=coverage.out -coverpkg=./internal/... ./tests/e2e/... ; \
	$(MAKE) firebase-emulator-stop

cover:
	go tool cover -html=coverage.out

firebase-pull:
	docker pull andreysenov/firebase-tools:$(FIREBASE_TOOLS_VERSION)

firebase-emulator: firebase-pull
	docker run -d --rm --name $(FIREBASE_CONTAINER_NAME) \
		-p $(FIREBASE_AUTH_PORT):9099 -p 4000:4000 -p 4400:4400 -p 4500:4500 \
		-v $(PWD)/firebase.json:/home/node/firebase.json \
		andreysenov/firebase-tools:$(FIREBASE_TOOLS_VERSION) \
		firebase emulators:start --only auth --project $(FIREBASE_PROJECT)

firebase-emulator-stop:
	docker stop $(FIREBASE_CONTAINER_NAME)

wait-firebase:
	@echo "Waiting for Firebase emulator..."
	@until curl -s http://localhost:$(FIREBASE_AUTH_PORT) > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Firebase emulator is ready"

.PHONY: build build-debian start deploy lint test-e2e firebase-pull firebase-emulator firebase-emulator-stop wait-firebase
