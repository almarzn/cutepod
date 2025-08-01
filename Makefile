# Build cute CLI
build:
	go build -o bin/cutepod ./main.go

# Generate CRDs using controller-gen
generate:
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:crdVersions=v1 paths=./internal/resource/... output:crd:artifacts:config=crds/
	go run sigs.k8s.io/controller-tools/cmd/controller-gen object paths=./internal/resource/...

# Run E2E test
e2e: build
	go test ./e2e -v

# Clean build artifacts
clean:
	rm -rf bin
