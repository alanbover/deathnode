VERSION?=$(shell git describe HEAD | sed s/^v//)
DOCKERNAME?=alanbover/deathnode
DOCKERTAG?=${DOCKERNAME}:${VERSION}

all: test build docker clean

build:
	mkdir -p docker/dist/
	CGO_ENABLED=0 GOOS=linux go build -o docker/dist/deathnode main.go 

docker:
	docker build -t ${DOCKERTAG} docker

test:
	go test $$(go list ./... | grep -v /vendor/)

clean:
	rm -rf docker/dist

lint:
	golint mesos/
	golint aws/
	golint deathnode/

