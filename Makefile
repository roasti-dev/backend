GO ?= go
FIREBASE := pnpm exec firebase
REDOCLY := pnpm exec redocly

GIT_COMMIT := $(shell git rev-parse --short HEAD)

FIREBASE_PROJECT := roasti-dev-project
FIREBASE_AUTH_PORT := 9099
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
		$(REDOCLY) lint $(OAPI_SPEC)

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

firebase-emulator:
	mkdir -p $(FIREBASE_DATA_DIR)
	@bash -c '\
		set -m; \
		echo $$$$ > .firebase.pid; \
		$(FIREBASE) emulators:start --only auth --project $(FIREBASE_PROJECT) \
			--import $(FIREBASE_DATA_DIR) & \
		FPID=$$!; \
		trap "$(FIREBASE) emulators:export $(FIREBASE_DATA_DIR) --project $(FIREBASE_PROJECT) --force; kill $$FPID; wait $$FPID; rm -f .firebase.pid; exit 0" INT TERM; \
		wait $$FPID \
	'

firebase-emulator-stop:
	@kill $$(cat .firebase.pid 2>/dev/null) 2>/dev/null || lsof -ti:$(FIREBASE_AUTH_PORT),4000,4400 | xargs kill 2>/dev/null || true
	@rm -f .firebase.pid

firebase-emulator-test:
	$(FIREBASE) emulators:start --only auth --project $(FIREBASE_PROJECT) & echo $$! > .firebase-test.pid

firebase-emulator-test-stop:
	kill $$(cat .firebase-test.pid) 2>/dev/null || true
	rm -f .firebase-test.pid

clean:
	rm -f $(DATABASE_PATH)
	rm -rf $(UPLOADS_PATH)
	rm -rf $(FIREBASE_DATA_DIR)

wait-firebase:
	@echo "Waiting for Firebase emulator..."
	@until curl -s http://localhost:$(FIREBASE_AUTH_PORT) > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Firebase emulator is ready"

.PHONY: build start clean setup-server deploy lint \
	test-e2e test-unit \
	firebase-emulator firebase-emulator-stop firebase-emulator-test firebase-emulator-test-stop \
	wait-firebase
