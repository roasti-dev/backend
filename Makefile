GO ?= go

GIT_COMMIT := $(shell git rev-parse --short HEAD)

FIREBASE_TOOLS_VERSION := 15.10.0@sha256:740d133bffbcda740b49f7e5ce883ecf7412752a931c68a6ad2040a0622e03a4
FIREBASE_PROJECT := roasti-dev-project
FIREBASE_AUTH_PORT := 9099
FIREBASE_CONTAINER_NAME := firebase-emulator-dev
FIREBASE_TEST_CONTAINER_NAME := firebase-emulator-test
FIREBASE_DATA_DIR := $(PWD)/.firebase-data

DATABASE_PATH  ?= data.db
UPLOADS_PATH   ?= ./uploads

OAPI_SPEC            := api/spec.yaml
OAPI_CONFIG          := api/spec-config.yaml
OAPI_MODELS_CONFIG   := api/models-config.yaml
OAPI_MODELS          := api/models.yaml
OAPI_OUT             := internal/handlers/server.gen.go
OAPI_MODELS_OUT      := internal/api/models/models.gen.go
OAPI_CLIENT_CONFIG   := api/client-config.yaml
OAPI_CLIENT_OUT      := tests/client/client.gen.go

OAPI_CODEGEN := $(GO) tool oapi-codegen

$(OAPI_MODELS_OUT): $(OAPI_MODELS) $(OAPI_MODELS_CONFIG)
	$(OAPI_CODEGEN) -config $(OAPI_MODELS_CONFIG) -o $@ $(OAPI_MODELS)

$(OAPI_OUT): $(OAPI_SPEC) $(OAPI_CONFIG) $(OAPI_MODELS_OUT)
	$(OAPI_CODEGEN) -config $(OAPI_CONFIG) -o $@ $(OAPI_SPEC)

$(OAPI_CLIENT_OUT): $(OAPI_SPEC) $(OAPI_CLIENT_CONFIG) $(OAPI_MODELS_OUT)
	$(OAPI_CODEGEN) -config $(OAPI_CLIENT_CONFIG) -o $@ $(OAPI_SPEC)

oapi: $(OAPI_MODELS_OUT) $(OAPI_OUT) $(OAPI_CLIENT_OUT)

build-%:
	$(eval PARTS := $(subst -, ,$*))
	$(eval GOOS  := $(word 1, $(PARTS)))
	$(eval GOARCH := $(word 2, $(PARTS)))
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build \
		-tags='no_postgres no_mysql no_mssql no_ydb no_vertica no_starrocks no_clickhouse no_turso' \
		-ldflags="-s -w -X main.appVersion=$(GIT_COMMIT)" \
		-o app-$(GOOS)-$(GOARCH) ./cmd/server

build:
	$(GO) build -o app ./cmd/server

start:
	APP_ENV=development DEBUG=true $(GO) run ./cmd/server

setup-server:
	ansible-playbook -i deploy/setup.ini deploy/setup.yaml

deploy:
	ansible-playbook -i deploy/deploy.ini deploy/deploy.yaml

lint:
	golangci-lint run

openapi-lint:
	REDOCLY_TELEMETRY=off REDOCLY_SUPPRESS_UPDATE_NOTICE=true \
		pnpm --package=@redocly/cli@2.22.1 dlx openapi lint $(OAPI_SPEC)

test-unit:
	$(GO) test ./internal/...

test-e2e: firebase-emulator-test wait-firebase
	APP_ENV=development \
	FIREBASE_PROJECT_ID=$(FIREBASE_PROJECT) \
	FIREBASE_API_KEY=test \
	FIREBASE_AUTH_EMULATOR_HOST=localhost:$(FIREBASE_AUTH_PORT) \
	FIREBASE_IDENTITY_BASE_URL=http://localhost:$(FIREBASE_AUTH_PORT)/identitytoolkit.googleapis.com/v1/accounts \
	FIREBASE_TOKEN_BASE_URL=http://localhost:$(FIREBASE_AUTH_PORT)/securetoken.googleapis.com/v1/token \
	$(GO) test -v -coverprofile=coverage.out -coverpkg=./internal/... ./tests/e2e/... ; \
	$(MAKE) firebase-emulator-test-stop

cover:
	$(GO) tool cover -html=coverage.out

firebase-pull:
	docker pull andreysenov/firebase-tools:$(FIREBASE_TOOLS_VERSION)

firebase-emulator: firebase-pull
	mkdir -p $(FIREBASE_DATA_DIR)
	docker run -d --rm --name $(FIREBASE_CONTAINER_NAME) \
		-p $(FIREBASE_AUTH_PORT):9099 -p 4000:4000 -p 4400:4400 -p 4500:4500 \
		-v $(PWD)/firebase.json:/home/node/firebase.json \
		-v $(FIREBASE_DATA_DIR):/data \
		andreysenov/firebase-tools:$(FIREBASE_TOOLS_VERSION) \
		firebase emulators:start --only auth --project $(FIREBASE_PROJECT) \
		--import /data --export-on-exit /data

firebase-emulator-test: firebase-pull
	docker run -d --rm --name $(FIREBASE_TEST_CONTAINER_NAME) \
		-p $(FIREBASE_AUTH_PORT):9099 -p 4000:4000 -p 4400:4400 -p 4500:4500 \
		-v $(PWD)/firebase.json:/home/node/firebase.json \
		andreysenov/firebase-tools:$(FIREBASE_TOOLS_VERSION) \
		firebase emulators:start --only auth --project $(FIREBASE_PROJECT)

dev: firebase-emulator wait-firebase
	FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:$(FIREBASE_AUTH_PORT) \
	FIREBASE_IDENTITY_BASE_URL=http://localhost:$(FIREBASE_AUTH_PORT)/identitytoolkit.googleapis.com/v1/accounts \
	FIREBASE_TOKEN_BASE_URL=http://localhost:$(FIREBASE_AUTH_PORT)/securetoken.googleapis.com/v1/token \
	DATABASE_PATH=$(DATABASE_PATH) \
	UPLOADS_PATH=$(UPLOADS_PATH) \
	SERVER_PORT=9090 DEBUG=1 $(GO) run ./cmd/server

clean:
	rm -f $(DATABASE_PATH)
	rm -rf $(UPLOADS_PATH)
	rm -rf $(FIREBASE_DATA_DIR)

firebase-emulator-stop:
	docker stop $(FIREBASE_CONTAINER_NAME)

firebase-emulator-test-stop:
	docker stop $(FIREBASE_TEST_CONTAINER_NAME)

wait-firebase:
	@echo "Waiting for Firebase emulator..."
	@until curl -s http://localhost:$(FIREBASE_AUTH_PORT) > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Firebase emulator is ready"

.PHONY: build build-debian start dev clean setup-server deploy lint \
	test-e2e test-unit firebase-pull \
	firebase-emulator firebase-emulator-stop \
	firebase-emulator-test firebase-emulator-test-stop \
	wait-firebase
