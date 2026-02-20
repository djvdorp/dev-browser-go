BIN      := dev-browser-go
CMD      := ./cmd/dev-browser-go
INSTALL  := $(HOME)/bin/$(BIN)

.PHONY: build install test vet smoke review tidy clean

## build: compile binary to repo root
build:
	go build -o $(BIN) $(CMD)

## install: build and copy to ~/bin
install:
	go build -o $(INSTALL) $(CMD)
	@echo "Installed $(INSTALL) ($$($(INSTALL) --version))"

## test: run unit tests
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## smoke: headless goto + snapshot (requires built binary)
smoke: build
	HEADLESS=1 ./$(BIN) goto https://example.com
	./$(BIN) snapshot

## review: Copilot diff review against main
review:
	./scripts/copilot_review_diff.sh

## tidy: tidy go.mod/go.sum
tidy:
	go mod tidy

## clean: remove local binary
clean:
	rm -f $(BIN)
