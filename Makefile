.PHONY: all build dist test install clean tools deps update-deps

all:
	@echo "build         - Build sup"
	@echo "dist          - Build sup distribution binaries"
	@echo "test          - Run tests"
	@echo "install       - Install binary"
	@echo "clean         - Clean up"
	@echo ""
	@echo "tools         - Install tools"
	@echo "vendor-list   - List vendor package tree"
	@echo "vendor-update - Update vendored packages"

build:
	@mkdir -p ./bin
	@rm -f ./bin/*
	go build -o ./bin/ssup ./cmd/ssup
	@rm -f $(HOME)/go/bin/ssup
	@cp -v ./bin/ssup $(HOME)/go/bin/ssup

dist:
	@mkdir -p ./bin
	@rm -f ./bin/*
	GOOS=darwin GOARCH=amd64 go build -o ./bin/ssup-darwin-amd64 ./cmd/sup
	GOOS=darwin GOARCH=arm64 go build -o ./bin/ssup-darwin-arm64 ./cmd/sup
	GOOS=linux GOARCH=amd64 go build -o ./bin/ssup-linux64 ./cmd/sup
	GOOS=linux GOARCH=386 go build -o ./bin/ssup-linux386 ./cmd/sup
	GOOS=windows GOARCH=amd64 go build -o ./bin/ssup-windows64.exe ./cmd/sup
	GOOS=windows GOARCH=386 go build -o ./bin/ssup-windows386.exe ./cmd/sup

test:
	go test ./...

install:
	go install ./cmd/sup

clean:
	@rm -rf ./bin

tools:
	go get -u github.com/kardianos/govendor

vendor-list:
	@govendor list

vendor-update:
	@govendor update +external
