.PHONY: build test test-pg measure vet clean

build:
	go build ./...

test:
	go test ./...

test-pg:
	go test -tags=integration ./...

vet:
	go vet ./...

clean:
	rm -rf bin/

measure:
	go test -tags=measure -run TestMeasure -v -timeout 0 ./...
