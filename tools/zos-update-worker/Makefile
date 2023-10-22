OUT=$(shell realpath -m bin)
GOPATH=$(shell go env GOPATH)
branch=$(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision=$(shell git rev-parse HEAD)
dirty=$(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
ldflags='-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty)'

all: getdeps test

getdeps:
	@echo "Installing gocyclo" && go get  -u github.com/fzipp/gocyclo/cmd/gocyclo
	@echo "Installing deadcode" && go get -u github.com/remyoudompheng/go-misc/deadcode
	@echo "Installing misspell" && go get -u github.com/client9/misspell/cmd/misspell
	@echo "Installing golangci-lint" && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin v1.50.1
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.50.1
	@wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.50.1


verifiers: fmt lint cyclo deadcode spelling staticcheck

fmt:
	@echo "Running $@"
	@gofmt -d .

lint:
	@echo "Running $@"
	@${GOPATH}/bin/golangci-lint run

cyclo:
	@echo "Running $@"
	@${GOPATH}/bin/gocyclo -over 100 .

deadcode:
	@echo "Running $@"
	@${GOPATH}/bin/deadcode -test $(shell go list ./...) || true

spelling:
	@${GOPATH}/bin/misspell -i monitord -error `find .`

staticcheck:
	go run honnef.co/go/tools/cmd/staticcheck -- ./...

test: verifiers
	go test -v -vet=off ./...

benchmarks: 
	go test -v -vet=off ./... -bench=. -count 1 -benchtime=10s -benchmem -run=^#

coverage: clean 
	mkdir coverage
	go test -v -vet=off ./... -coverprofile=coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html

testrace: verifiers
	go test -v -race -vet=off ./...

run: 
	go run main.go

build:
	go build -o bin/zos-update-worker main.go 
	
clean:
	rm ./coverage -rf
	rm ./bin -rf
