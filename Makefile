OUT=$(shell realpath -m bin)
GOPATH=$(shell go env GOPATH)
branch=$(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
revision=$(shell git rev-parse HEAD)
dirty=$(shell test -n "`git diff --shortstat 2> /dev/null | tail -n1`" && echo "*")
ldflags='-w -s -X $(version).Branch=$(branch) -X $(version).Revision=$(revision) -X $(version).Dirty=$(dirty)'

all: getdeps test

getdeps:
	@echo "Installing golint" && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.0
	@echo "Installing gocyclo" && go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@echo "Installing deadcode" && go install github.com/remyoudompheng/go-misc/deadcode@latest
	@echo "Installing misspell" && go install github.com/client9/misspell/cmd/misspell@latest
	@echo "Installing staticcheck" && go install honnef.co/go/tools/cmd/staticcheck@latest


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
	@${GOPATH}/bin/staticcheck -- ./...

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

run-worker: 
	go run ./tools/zos-update-worker/main.go

build-worker:
	go build -o ./tools/zos-update-worker/bin/zos-update-worker ./tools/zos-update-worker/main.go 
	
clean:
	rm ./coverage -rf
	rm ./bin -rf
