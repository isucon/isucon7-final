GOPATH := ${PWD}
export GOPATH

build:
		go build -v bench

ensure:
		cd src/bench && dep ensure -vendor-only

update:
		cd src/bench && dep ensure

race:
		go build -v -race bench

vet:
	go vet ./src/...
