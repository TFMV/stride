.PHONY: all build test clean install install-user install-system

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY_NAME=stride
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

all: test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd

test:
	$(GOTEST) -v ./...

clean:
	rm -f $(BINARY_NAME)
	rm -f $(HOME)/go/bin/$(BINARY_NAME)
	rm -f /usr/local/bin/$(BINARY_NAME)

# Install to user's go/bin directory
install-user: build
	mkdir -p $(HOME)/go/bin
	cp $(BINARY_NAME) $(HOME)/go/bin/
	@echo "Installed $(BINARY_NAME) to $(HOME)/go/bin/"
	@echo "Make sure $(HOME)/go/bin is in your PATH"

# Install system-wide (requires sudo)
install-system: build
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

# Default install is user-level
install: install-user 