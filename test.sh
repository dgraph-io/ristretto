#! /bin/sh
DEFAULT_VERSION="1.17"
GOVERSION=$({ [ -f .go-version ] && cat .go-version; } || echo $DEFAULT_VERSION)

starttest() {
	set -e
	GO111MODULE=on go test -race ./...
}

if [ -z "${TEAMCITY_VERSION}" ]; then
	# running locally, so start test in a container
	# TEAMCITY_VERSION=local will avoid recursive calls, when it would be running in container
	docker run --rm --name ristretto-test --tty --interactive \
	  --volume `pwd`:/go/src/github.com/dgraph-io/ristretto \
  	--workdir /go/src/github.com/dgraph-io/ristretto \
		--env TEAMCITY_VERSION=local \
  	golang:$GOVERSION \
  	sh test.sh
else
	# running in teamcity, since teamcity itself run this in container, let's simply run this
	starttest
fi
