BINARY := skimd
CMD := ./cmd/skimd
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

.PHONY: build install test test-race vet check clean print-tmux-binding

build:
	go build -o $(BINARY) $(CMD)

install: build
	mkdir -p $(BINDIR)
	install -m 755 $(BINARY) $(BINDIR)/$(BINARY)

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

check: test test-race vet

clean:
	rm -f $(BINARY)

print-tmux-binding: build
	./$(BINARY) --print-tmux-binding
