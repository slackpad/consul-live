SHELL = bash

GOFILES ?= $(shell go list ./... | grep -v /vendor/)

test: vet
	@echo "--> Running go test"
	go test ./...

bin:
	mkdir -p bin/
	GOOS=linux GOARCH=amd64 go build -o bin/consul-live

pkg: bin
	mkdir -p pkg/
	tar -czf pkg/consul-live.tar.gz -C bin/ .

test-race:
	$(MAKE) GOTEST_FLAGS=-race

cover:
	go test $(GOFILES) --cover

format:
	@echo "--> Running go fmt"
	@go fmt $(GOFILES)

vet:
	@echo "--> Running go vet"
	@go vet $(GOFILES); if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

.PHONY: bin pkg test test-race cover format vet
