BINARY := tact

.PHONY: build install clean release

build:
	go build -o $(BINARY) ./cmd/tact/

install:
	go install ./cmd/tact/

clean:
	rm -f $(BINARY)

release:
	goreleaser release --clean
