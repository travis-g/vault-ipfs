fmt:
	@echo "==> Running gofmt..."
	gofmt -s -w .

build: fmt
	@echo "==> Building library..."
	go build -ldflags="-s -w" ./...
	@echo "==> Building the CLI..."
	go build -ldflags="-s -w" ./cmd/vault-plugin-ipfs

.PHONY: build fmt
