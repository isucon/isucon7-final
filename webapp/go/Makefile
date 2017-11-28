GOPATH := ${PWD}
export GOPATH

build:
		go build -v app

ensure:
		cd src/app && dep ensure -vendor-only

update:
		cd src/app && dep ensure

test:
		go test -v app

vet:
		go vet ./src/app/...
