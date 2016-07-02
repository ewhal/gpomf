## simple makefile to log workflow
.PHONY: all test clean build install

GOFLAGS ?= $(GOFLAGS:)

all: clean install build

build:
	go build $(GOFLAGS) ./...

install:
	go get github.com/go-sql-driver/mysql 
	go get github.com/dchest/uniuri 

test: install
	go test $(GOFLAGS) ./...

bench: install
	go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	go clean $(GOFLAGS) -i ./...
	rm -rf ./build 
