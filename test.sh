#! /bin/sh

starttest() {
	set -e
	GO111MODULE=on go test -race ./...
}

if [ -z "${TEAMCITY_VERSION}" ]; then
	docker run --rm --name ristretto-test -ti \
  		-v `pwd`:/go/src/github.com/dgraph-io/ristretto \
  		--workdir /go/src/github.com/dgraph-io/ristretto \
		--env TEAMCITY_VERSION=local \
  		golang:1.13 \
  		sh test.sh
else
	starttest
fi
