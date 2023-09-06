
.PHONY: install test build

CURRENT_DIR := $(shell pwd)

build:
	@echo "Building GoBC..."
	go build -v -o bin/gobc cmd/gobc/gobc.go

install:
	@echo "Installing GoBC..."
	go install -v ./cmd/gobc/gobc.go

cpu_instr: build
	@echo "Running CPU instrs..."
	$(CURRENT_DIR)/bin/gobc --panic-on-stuck $(CURRENT_DIR)/default_rom/blarrg/cpu_instrs/cpu_instrs.gb

instr_timing: build
	@echo "Running instr timing..."
	$(CURRENT_DIR)/bin/gobc --panic-on-stuck $(CURRENT_DIR)/default_rom/blarrg/instr_timing/instr_timing.gb

test: build  cpu_instr instr_timing