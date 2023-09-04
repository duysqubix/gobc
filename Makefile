
.PHONY: install test build

build:
	@echo "Building GoBC..."
	go build -v -o bin/gobc cmd/gobc/gobc.go

install:
	@echo "Installing GoBC..."
	go install -v ./cmd/gobc/gobc.go

test: build  
	@echo "Running tests..."
	LOG_LEVEL=warn bin/gobc /home/duys/.repos/gobc/default_rom/blarrg/instr_timing/instr_timing.gb | /bin/grep -q "Passed"
	LOG_LEVEL=warn bin/gobc /home/duys/.repos/gobc/default_rom/blarrg/cpu_instrs/cpu_instrs.gb | /bin/grep -q "Passed"