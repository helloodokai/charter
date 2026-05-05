.PHONY: build test lint vet schema install clean

BINARY=charter
GO=go
LDFLAGS=-ldflags "-s -w"

build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY) ./cmd/charter

build-app:
	$(GO) build $(LDFLAGS) -o bin/charter-app ./cmd/charter-app

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

vet:
	$(GO) vet ./...

schema: bin
	$(GO) run ./cmd/charter schema --out schema/charter.schema.json

install: build
	cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY) 2>/dev/null || sudo cp bin/$(BINARY) /usr/local/bin/$(BINARY)

bin:
	mkdir -p bin

clean:
	rm -rf bin/

all: build build-app