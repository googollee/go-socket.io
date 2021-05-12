all:
	go install ./...

get_dev:
	go get -t ./...

test:
	go clean -testcache && go test -race -cover -covermode=atomic ./...

bench:
	go clean -testcache && go test -bench . -benchmem ./...

lint:
	golangci-lint run --config=.golangci.yml ./...

cover:
	go clean -testcache && go test ./... -cover -coverprofile=c.out && go tool cover -html=c.out
