.PHONY: banner build test lint tidy demo vhs

banner:
	cp scripts/banner.ansi internal/cli/banner.ansi

build: banner
	go build -o bin/pccli ./cmd/pccli

test:
	go test ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

demo:
	PCCLI_MOCK=true go run ./cmd/pccli services songs list

vhs: build
	PATH="$(PWD)/bin:$(PATH)" vhs scripts/demo.tape
