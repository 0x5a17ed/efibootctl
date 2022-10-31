BINARY_NAME := efibootctl

GO_LDFLAGS=-ldflags "-s -w -extldflags '-static'"

SRC = $(shell find . -type f -name '*.go')

dist/$(BINARY_NAME): $(SRC) go.mod go.sum
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v2 go build -trimpath $(GO_LDFLAGS) -o $@

dist/$(BINARY_NAME).exe: $(SRC) go.mod go.sum
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 GOAMD64=v2 go build -trimpath $(GO_LDFLAGS) -o $@

.PHONY: all
all: dist/$(BINARY_NAME) dist/$(BINARY_NAME).exe

.PHONY: clean
clean:
	-rm dist/$(BINARY_NAME) dist/$(BINARY_NAME).exe
