# Copyright © 2019, Oleksandr Krykovliuk <k33nice@gmail.com>.
# Use of this source code is governed by the
# MIT license that can be found in the LICENSE file.

SHELL = bash

GO ?= go
arch ?= amd64

.PHONY: all
all: all-tests
	@echo "Done!"

.PHONY: get
get:
	@echo "Resolve dependencies..."
	@$(GO) get -v .

.PHONY: all-tests
all-tests:
	@echo "Run tests..."
	@$(GO) test .

.PHONY: benchmark
benchmark:
	@echo "Run benchmarks..."
	@$(GO) test -v -benchmem -bench=. -run=^a

.PHONY: test-race
test-race:
	@echo "Run tests with race condition..."
	@$(GO) test --race -v .

.PHONY: test-cover-builder
test-cover-builder:
	@$(GO) test -covermode=count -coverprofile=/tmp/libart.out .

	@rm -f /tmp/art_coverage.out
	@echo "mode: count" > /tmp/art_coverage.out
	@cat /tmp/libart.out | tail -n +2 >> /tmp/art_coverage.out
	@rm /tmp/libart.out

.PHONY: test-cover
test-cover: test-cover-builder
	@$(GO) tool cover -html=/tmp/art_coverage.out

.PHONY: build
build:
	@echo "Build project..."
	@$(GO) build -v .

.PHONY: build-asm
build-asm:
	@$(GO) build -a -work -v -gcflags="-S -B -C" .

.PHONY: build-race
build-race:
	@echo "Build project with race condition..."
	@$(GO) build --race -v .
