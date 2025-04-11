BINARY_NAME=picalc
VERSION=$(shell grep -m1 "VERSION =" pkg/picalc/picalc.go | cut -d '"' -f2)
BUILD_DIR=build
MAIN_PATH=cmd/picalc
PKG_PATH=pkg/picalc
DOCKER_REPO=ghcr.io/yourusername/picalc

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOINSTALL=$(GOCMD) install

LDFLAGS=-ldflags "-X main.version=$(VERSION) -s -w"
BENCH_FLAGS=-benchmem -benchtime=10s
RACE_FLAGS=-race
COVER_FLAGS=-coverprofile=coverage.out
PROF_FLAGS=-cpuprofile=cpu.prof -memprofile=mem.prof

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    OPEN_CMD=open
else
    OPEN_CMD=xdg-open
endif

.PHONY: all build clean test bench cover perf race install update-deps lint docker docker-run help release

all: clean build test

build:
	@echo "Building PiCalc v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(MAIN_PATH)
	@echo "Binary created at $(BUILD_DIR)/$(BINARY_NAME)"

build-all: clean
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	@echo "Building for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_linux_amd64 ./$(MAIN_PATH)
	
	@echo "Building for Linux (arm64)..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_linux_arm64 ./$(MAIN_PATH)
	
	@echo "Building for macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64 ./$(MAIN_PATH)
	
	@echo "Building for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_arm64 ./$(MAIN_PATH)
	
	@echo "Building for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_windows_amd64.exe ./$(MAIN_PATH)
	
	@echo "All binaries created in $(BUILD_DIR)/"

clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out cpu.prof mem.prof
	@echo "Cleaned!"

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

race:
	@echo "Running tests with race detector..."
	$(GOTEST) $(RACE_FLAGS) ./...

bench:
	@echo "Running benchmarks..."
	$(GOTEST) $(BENCH_FLAGS) -bench=. ./$(PKG_PATH)

cover:
	@echo "Generating test coverage report..."
	$(GOTEST) $(COVER_FLAGS) ./...
	$(GOCMD) tool cover -html=coverage.out
	@echo "Coverage report generated"

perf:
	@echo "Running performance profiling..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) $(PROF_FLAGS) -bench=. ./$(PKG_PATH)
	@echo "CPU profile saved to cpu.prof"
	@echo "Memory profile saved to mem.prof"
	@echo "View profiles with: go tool pprof cpu.prof"

install:
	@echo "Installing PiCalc..."
	$(GOINSTALL) $(LDFLAGS) ./$(MAIN_PATH)
	@echo "Installed at $(shell which $(BINARY_NAME))"

update-deps:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy
	@echo "Dependencies updated!"

lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
		golangci-lint run ./...; \
	fi

docker:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_REPO):$(VERSION) -t $(DOCKER_REPO):latest .
	@echo "Docker image built: $(DOCKER_REPO):$(VERSION)"

docker-run:
	@echo "Running PiCalc in Docker..."
	docker run --rm $(DOCKER_REPO):latest calculate 100

release: build-all
	@echo "Preparing release v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)/release
	
	@echo "Creating Linux archives..."
	tar -czf $(BUILD_DIR)/release/$(BINARY_NAME)_$(VERSION)_linux_amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)_linux_amd64
	tar -czf $(BUILD_DIR)/release/$(BINARY_NAME)_$(VERSION)_linux_arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)_linux_arm64
	
	@echo "Creating macOS archives..."
	tar -czf $(BUILD_DIR)/release/$(BINARY_NAME)_$(VERSION)_darwin_amd64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)_darwin_amd64
	tar -czf $(BUILD_DIR)/release/$(BINARY_NAME)_$(VERSION)_darwin_arm64.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)_darwin_arm64
	
	@echo "Creating Windows archive..."
	cd $(BUILD_DIR) && zip -r release/$(BINARY_NAME)_$(VERSION)_windows_amd64.zip $(BINARY_NAME)_windows_amd64.exe
	
	@echo "Release artifacts created in $(BUILD_DIR)/release/"

demo: build
	@echo "Running a quick π calculation demo..."
	@$(BUILD_DIR)/$(BINARY_NAME) calculate 1000

view-cpu-profile: perf
	@echo "Opening CPU profile in browser..."
	$(GOCMD) tool pprof -http=:8080 cpu.prof

view-mem-profile: perf
	@echo "Opening memory profile in browser..."
	$(GOCMD) tool pprof -http=:8081 mem.prof

help:
	@echo "PiCalc Makefile Commands:"
	@echo "-------------------------"
	@echo "make                 - Clean, build, and test"
	@echo "make build           - Build for current platform"
	@echo "make build-all       - Build for multiple platforms"
	@echo "make clean           - Remove build artifacts"
	@echo "make test            - Run tests"
	@echo "make race            - Run tests with race detector"
	@echo "make bench           - Run benchmarks"
	@echo "make cover           - Generate and view test coverage"
	@echo "make perf            - Run performance profiling"
	@echo "make install         - Install binary to GOPATH"
	@echo "make update-deps     - Update dependencies"
	@echo "make lint            - Run linter"
	@echo "make docker          - Build Docker image"
	@echo "make docker-run      - Run from Docker"
	@echo "make release         - Build release artifacts"
	@echo "make demo            - Run a quick demo"
	@echo "make view-cpu-profile - View CPU profile in browser"
	@echo "make view-mem-profile - View memory profile in browser"
	@echo "make help            - Display this help"
	@echo ""
	@echo "Current version: v$(VERSION)"
