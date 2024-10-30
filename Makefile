VERSION != git describe --tags --abbrev=0
REVISION != git rev-parse --short HEAD

.PHONY: build
build:
	go build -ldflags "-X github.com/t-kuni/sisho/cmd/versionCommand.Version=$(VERSION) -X github.com/t-kuni/sisho/cmd/versionCommand.Revision=$(REVISION)" main.go

.PHONY: test
test: generate
	go test -v ./...

generate:
	go generate ./...