NAME      := terraform-provider-scalegrid
VERSION   ?= dev
HOSTNAME  := registry.terraform.io
NAMESPACE := requestflo
PROVIDER  := scalegrid
OS_ARCH   := $(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_DIR := ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(PROVIDER)/$(VERSION)/$(OS_ARCH)

.PHONY: default build install test testacc fmt vet lint tidy clean

default: build

## build: compile the provider binary
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(NAME)

## install: build and install the provider into the local plugin cache for testing
install: build
	mkdir -p $(INSTALL_DIR)
	cp $(NAME) $(INSTALL_DIR)/$(NAME)_v$(VERSION)

## test: run unit tests
test:
	go test ./... -timeout 120s

## testacc: run acceptance tests (requires a live ScaleGrid account)
testacc:
	TF_ACC=1 go test ./... -v -timeout 120m

## fmt: format Go sources
fmt:
	gofmt -w .

## vet: run go vet
vet:
	go vet ./...

## tidy: tidy go modules
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -f $(NAME)
