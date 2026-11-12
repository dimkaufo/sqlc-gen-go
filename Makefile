.PHONY: build test test.update release

build:
	go build ./...

test: bin/sqlc-gen-go.wasm
	go test ./...
	cd internal/endtoend && go test ./...

# Regenerate the nested end-to-end case's checked-in output after an intended generator change.
test.update:
	cd internal/endtoend && go test -run TestNestedGenerate . -update

all: bin/sqlc-gen-go bin/sqlc-gen-go.wasm

# auto: use go.mod toolchain (download if newer than local); local breaks when go.mod > installed Go.
bin/sqlc-gen-go: bin go.mod go.sum $(wildcard **/*.go)
	cd plugin && GOTOOLCHAIN=auto go build -o ../bin/sqlc-gen-go ./main.go

bin/sqlc-gen-go.wasm: bin/sqlc-gen-go
	cd plugin && GOTOOLCHAIN=auto GOOS=wasip1 GOARCH=wasm go build -o ../bin/sqlc-gen-go.wasm main.go

bin:
	mkdir -p bin

# Build the WASM release and print the steps to publish it (interactive; VERSION=vX.Y.Z[-rc.N]).
release:
	@VERSION="$(VERSION)" ./scripts/release.sh