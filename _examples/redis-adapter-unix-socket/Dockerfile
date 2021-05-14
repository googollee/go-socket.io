FROM golang:1.16-alpine AS build

WORKDIR /go/src/app
COPY main.go go.mod

RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -o /usr/local/bin/app

FROM scratch
COPY --from=build /usr/local/bin/app /usr/local/bin/app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]
