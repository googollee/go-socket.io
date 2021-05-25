
IMAGE_NAME=default-http
CONTAINER_NAME=default-http

all: run

run: build
	docker run -d -p 8000:8000 --name $(CONTAINER_NAME) $(IMAGE_NAME):latest

build: clean_up
	docker build . -t $(IMAGE_NAME)

clean_up: stop
	docker rm $(CONTAINER_NAME)
	docker rmi $(IMAGE_NAME)

stop:
	docker stop $(CONTAINER_NAME)