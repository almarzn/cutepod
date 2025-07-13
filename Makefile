# Build cute CLI
build:
	go build -o bin/cutepod ./main.go

# Run E2E test
e2e: build
	go test ./e2e -v

# Clean build artifacts
clean:
	rm -rf bin
