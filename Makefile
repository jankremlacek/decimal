# Mutation-fuzzing budget per fuzz target (override via make var FUZZTIME).
FUZZTIME?=2m

.PHONY: \
	bench \
	help \
	test \
	test-fuzz \
	vet

## help: Prints this help message.
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## vet: Run go vet analyzers
vet:
	@echo "Running go vet ..."
	@go vet ./...

## test: Run the tests with coverage (replays the pinned fuzz seeds, does not fuzz)
test:
	@go test -race -cover ./...

## test-fuzz: Run every fuzzer in mutation mode, FUZZTIME each (go allows one -fuzz per invocation)
test-fuzz:
	@go test . -run '^$$' -fuzz '^FuzzNewFromString$$' -fuzztime $(FUZZTIME)
	@go test . -run '^$$' -fuzz '^FuzzArithmetic$$' -fuzztime $(FUZZTIME)

## bench: Run the benchmarks, including the side-by-side shopspring comparison
bench:
	@go test . -run '^$$' -bench . -benchmem
