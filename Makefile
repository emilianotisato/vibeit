PREFIX ?= ~/.local
BINARY = vibeit

.PHONY: build install clean run doctor

build:
	go build -o $(BINARY) ./cmd/vibeit

install: build
	mkdir -p $(PREFIX)/bin
	cp $(BINARY) $(PREFIX)/bin/$(BINARY).tmp
	mv -f $(PREFIX)/bin/$(BINARY).tmp $(PREFIX)/bin/$(BINARY)

uninstall:
	rm -f $(PREFIX)/bin/$(BINARY)

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY)

doctor: build
	./$(BINARY) doctor

# Development helpers
fmt:
	go fmt ./...

lint:
	golangci-lint run

test:
	go test ./...
