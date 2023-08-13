
.PHONY: run test


run:
	@echo "Running GoBC..."
	go run cmd/gobc/main.go

test: 
	go test ./...