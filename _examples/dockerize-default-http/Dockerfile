FROM golang:1.16-alpine as build

ENV GO11MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    BASE_PATH=/go/src/app

WORKDIR $BASE_PATH

COPY . .

RUN go mod download
RUN go build -o main .


# Run section
FROM scratch as run

ENV BASE_PATH=/go/src/app

COPY --from=build $BASE_PATH/main /app/main
COPY --from=build $BASE_PATH/asset /asset

EXPOSE 8000

ENTRYPOINT ["/app/main"]
