# Makefile for badgerdb-sampler

.PHONY: all build-all build-v2 build-v3 build-v4 clean

# Default target
all: build-all

# Build all versions
build-all: build-v2 build-v3 build-v4

# Build BadgerDB v2 version
build-v2:
	@echo "Building badgerdb-sampler-v2..."
	@cp go_v2.mod go.mod && cp go_v2.sum go.sum 2>/dev/null || true
	go build -tags badgerv2 -o bin/badgerdb-sampler-v2
	@echo "✓ Built: bin/badgerdb-sampler-v2"

# Build BadgerDB v3 version
build-v3:
	@echo "Building badgerdb-sampler-v3..."
	@cp go_v3.mod go.mod && cp go_v3.sum go.sum 2>/dev/null || true
	go build -tags badgerv3 -o bin/badgerdb-sampler-v3
	@echo "✓ Built: bin/badgerdb-sampler-v3"

# Build BadgerDB v4 version
build-v4:
	@echo "Building badgerdb-sampler-v4..."
	@cp go_v4.mod go.mod && cp go_v4.sum go.sum 2>/dev/null || true
	go build -tags badgerv4 -o bin/badgerdb-sampler-v4
	@echo "✓ Built: bin/badgerdb-sampler-v4"

# Clean build artifacts
clean:
	@echo "Cleaning bin/..."
	rm -f bin/badgerdb-sampler-v2 bin/badgerdb-sampler-v3 bin/badgerdb-sampler-v4
	@echo "Cleaning outputs/..."
	rm -rf outputs/*
	@echo "✓ Cleaned"
