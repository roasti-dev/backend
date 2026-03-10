GO ?= go

STANDARD_ENUMS := ./internal/recipe/brew_method.go \
	./internal/recipe/difficulty.go

NULLABLE_ENUMS = ./internal/recipe/roast_level.go

STANDARD_ENUMS_GO := $(STANDARD_ENUMS:.go=_enum.go)
NULLABLE_ENUMS_GO := $(NULLABLE_ENUMS:.go=_enum.go)

$(STANDARD_ENUMS_GO): GO_ENUM_FLAGS=--marshal --names --sqlint
$(NULLABLE_ENUMS_GO): GO_ENUM_FLAGS=--marshal --names --sqlnullint

enums: $(STANDARD_ENUMS_GO) $(NULLABLE_ENUMS_GO)

%_enum.go: %.go
	$(GO) tool go-enum -f $< $(GO_ENUM_FLAGS)

build:
	$(GO) build -o app ./cmd/server

start:
	$(GO) run ./cmd/server

.PHONY: build start