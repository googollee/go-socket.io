
IMAGE_NAME=gin-redis-adapter
CONTAINER_NAME=gin-redis-adapter

all: run

run: build
	docker-compose up -d

build: clean_up
	docker build . -t $(IMAGE_NAME)

clean_up: stop
	docker-compose down
	docker rm $(CONTAINER_NAME)
	docker rmi $(IMAGE_NAME)

stop:
	docker stop $(CONTAINER_NAME)