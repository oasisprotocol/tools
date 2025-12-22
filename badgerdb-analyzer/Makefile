# Makefile for badgerdb-analyzer

.PHONY: all build-all build-v2 build-v3 build-v4 clean tidy-all

# Default target
all: build-all

# Build all versions
build-all: build-v2 build-v3 build-v4

# Build BadgerDB v2 version
build-v2:
	@echo "Building badgerdb-analyzer-v2..."
	go build -modfile=go_v2.mod -tags badgerv2 -o bin/badgerdb-analyzer-v2
	@echo "✓ Built: bin/badgerdb-analyzer-v2"

# Build BadgerDB v3 version
build-v3:
	@echo "Building badgerdb-analyzer-v3..."
	go build -modfile=go_v3.mod -tags badgerv3 -o bin/badgerdb-analyzer-v3
	@echo "✓ Built: bin/badgerdb-analyzer-v3"

# Build BadgerDB v4 version
build-v4:
	@echo "Building badgerdb-analyzer-v4..."
	go build -modfile=go_v4.mod -tags badgerv4 -o bin/badgerdb-analyzer-v4
	@echo "✓ Built: bin/badgerdb-analyzer-v4"

# Clean build artifacts
clean:
	@echo "Cleaning bin/..."
	rm -f bin/badgerdb-sampler-v2 bin/badgerdb-sampler-v3 bin/badgerdb-sampler-v4
	@echo "Cleaning outputs/..."
	rm -rf outputs/*
	@echo "✓ Cleaned"

# Tidy all module files
tidy-all:
	@echo "Tidying all module files..."
	GOFLAGS="-tags=badgerv2" go mod tidy -modfile=go_v2.mod
	GOFLAGS="-tags=badgerv3" go mod tidy -modfile=go_v3.mod
	GOFLAGS="-tags=badgerv4" go mod tidy -modfile=go_v4.mod
	@echo "✓ All module files tidied"
