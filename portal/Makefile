all: build

deps:
	gb vendor restore

src/portal/bindata.go: views/*.tmpl static/*
	go-bindata  -pkg portal -o src/portal/bindata.go $^

.PHONY: build
build: src/portal/bindata.go
	GOPATH=`pwd`:`pwd`/vendor go install ./src/cmd/...

.PHONY: race
race: src/portal/bindata.go
	GOPATH=`pwd`:`pwd`/vendor go install -race ./src/cmd/...
