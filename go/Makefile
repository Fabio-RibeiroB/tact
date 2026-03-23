BINARY := tact

.PHONY: build install clean release check test vet

build:
	go build -o $(BINARY) ./cmd/tact/

install:
	go install ./cmd/tact/

clean:
	rm -f $(BINARY)

release:
	goreleaser release --clean

check: vet test

vet:
	go vet ./...

test:
	go test ./...
