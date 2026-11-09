.PHONY: build test vet lint release-dry

build:
	go build -o pushall ./cmd/pushall

test:
	go test ./...

vet:
	go vet ./...

lint:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	go vet ./...

release-dry:
	goreleaser release --snapshot --clean
