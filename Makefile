
.PHONY: run test build

build:
	@echo "Building GoBC..."
	go build -v -o gobc cmd/gobc/gobc.go

run:
	@echo "Running GoBC..."
	go run cmd/gobc/main.go

test: 
	go test ./...