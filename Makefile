GO ?= go

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
OAPI_OUT := internal/api/server.gen.go
OAPI_MODELS_OUT := internal/api/models/models.gen.go

$(OAPI_MODELS_OUT): $(OAPI_MODELS) $(OAPI_MODELS_CONFIG)
	$(GO) tool oapi-codegen -config $(OAPI_MODELS_CONFIG) -o $(OAPI_MODELS_OUT) $(OAPI_MODELS)

$(OAPI_OUT): $(OAPI_SPEC) $(OAPI_CONFIG) $(OAPI_MODELS_OUT)
	$(GO) tool oapi-codegen -config $(OAPI_CONFIG) -o $(OAPI_OUT) $(OAPI_SPEC)

oapi-gen: $(OAPI_MODELS_OUT) $(OAPI_OUT)

oapi: $(OAPI_OUT)

build:
	$(GO) build -o app ./cmd/server

# Debian 13 (Trixie) 64-bit
build-debian:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		$(GO) build -ldflags="-s -w" -o $(OUTPUT_DEBIAN) ./cmd/server

start:
	DEBUG=true $(GO) run ./cmd/server

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
	ssh $(REMOTE) '\
		mv $(REMOTE_BIN).new $(REMOTE_BIN) && \
		chmod +x $(REMOTE_BIN) && \
		sudo systemctl restart $(BACKEND_SERVICE) && \
		sudo systemctl status $(BACKEND_SERVICE) --no-pager \
	'

.PHONY: build start deploy