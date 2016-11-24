version=0.1.0

.PHONY: all

NOVENDOR := $(shell glide novendor)

build: clean
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/worker ./worker

clean:
	@rm -rf ./bin

lint:
	@go vet $(NOVENDOR)
	@go list ./... | grep -v /vendor/ | xargs -L1 golint

test:
	@go test $(NOVENDOR)

docker: build
	@docker build -t rannu-worker:latest .
