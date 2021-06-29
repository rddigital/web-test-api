.PHONY: build test clean docker docker-push docker-arm64

GO=CGO_ENABLED=1 go
OUTPUT=web

build:
	$(GO) build -o $(OUTPUT)

run:
	$(GO) run main.go


