BIN      := dev-browser-go
CMD      := ./cmd/dev-browser-go
INSTALL  := $(HOME)/bin/$(BIN)
GOFILES  := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './.git/*' | sort)
GOFUMPT  ?= $(shell command -v gofumpt 2>/dev/null)

ifeq ($(GOFUMPT),)
GOFUMPT := go run mvdan.cc/gofumpt@latest
endif

.PHONY: build install fmt fmt-check fumpt fumpt-check lint test vet docs-audit ci smoke review tidy clean

## build: compile binary to repo root
build:
	go build -o $(BIN) $(CMD)

## install: build and copy to ~/bin
install:
	go build -o $(INSTALL) $(CMD)
	@echo "Installed $(INSTALL) ($$($(INSTALL) --version))"

## fmt: format Go files with gofmt
fmt:
	gofmt -w $(GOFILES)

## fmt-check: fail if Go files are not gofmt formatted
fmt-check:
	@out="$$(gofmt -l $(GOFILES))"; \
	if [ -n "$$out" ]; then \
		echo "gofmt mismatch:"; \
		echo "$$out"; \
		exit 1; \
	fi

## fumpt: format Go files with gofumpt
fumpt:
	$(GOFUMPT) -w $(GOFILES)

## fumpt-check: fail if Go files are not gofumpt formatted
fumpt-check:
	@out="$$( $(GOFUMPT) -l $(GOFILES) )"; \
	if [ -n "$$out" ]; then \
		echo "gofumpt mismatch:"; \
		echo "$$out"; \
		exit 1; \
	fi

## lint: lightweight static checks
lint: fmt-check fumpt-check vet docs-audit

## test: run unit tests
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## docs-audit: verify README/SKILL match the current CLI surface
docs-audit:
	./scripts/check_docs_surface.sh

## ci: local CI sequence
ci: fmt-check fumpt-check vet docs-audit test build

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
