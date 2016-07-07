version=0.2.0
name=rannu
vcs=github.com/unchartedsoftware/$(name)

.PHONY: all

build: clean
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./bin/main .

clean:
	@rm -rf ./bin

lint:
	@go vet ./...
	@golint ./...

docker: build
	@docker build -t rannu:latest .
