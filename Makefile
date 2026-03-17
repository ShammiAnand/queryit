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

INSTALL_CMD := install
ifeq ($(shell test -w $(BINDIR) 2>/dev/null || echo no), no)
	INSTALL_CMD := sudo install
endif

install: build
	$(INSTALL_CMD) -d $(BINDIR)
	$(INSTALL_CMD) -m 755 $(BINARY) $(BINDIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(BINDIR)/$(BINARY)"

uninstall:
	$(INSTALL_CMD) -d $(BINDIR)
	rm -f $(BINDIR)/$(BINARY) 2>/dev/null || sudo rm -f $(BINDIR)/$(BINARY)
	@echo "Removed $(BINDIR)/$(BINARY)"

clean:
	rm -f $(BINARY)

clean-cache:
	rm -rf $${XDG_CACHE_HOME:-$$HOME/.cache}/queryit
	@echo "Cache cleared"

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

run: build
	./$(BINARY)
