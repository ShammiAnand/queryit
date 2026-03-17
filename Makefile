BINARY     := queryit
MODULE     := $(shell go list -m)
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(BUILD_DATE)

PREFIX     ?= /usr/local
BINDIR     := $(PREFIX)/bin

.PHONY: all build install uninstall clean fmt vet lint test run

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install: build
	install -d $(BINDIR)
	install -m 755 $(BINARY) $(BINDIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(BINDIR)/$(BINARY)"

uninstall:
	rm -f $(BINDIR)/$(BINARY)
	@echo "Removed $(BINDIR)/$(BINARY)"

clean:
	rm -f $(BINARY)

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

run: build
	./$(BINARY)
