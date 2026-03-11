GO ?= go

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

start:
	$(GO) run ./cmd/server

.PHONY: build start