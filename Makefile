IMAGE_REPOSITORY = localhost:5001
IMAGE_VERSION = latest

.PHONY: test

all: build

build:
	cd mc-ytt-bridge && go build -o mc-ytt-bridge main.go

run:
	cd mc-ytt-bridge && go run main.go serve --handlers test/handlers --config test/config.yaml

test: unit-tests

unit-tests:
	cd mc-ytt-bridge && go test pkg/bridge/*_test.go -v

cmd-tests: test-echo test-values

test-echo:
	curl -v -d @test/requests/simple.json -H Content-Type:application/json 'http://localhost:8080/testing/tests/echo?arg1=val1&arg2=val2'

test-values:
	curl -v -d @test/requests/simple.json -H Content-Type:application/json 'http://localhost:8080/testing/tests/values?arg1=val1&arg2=val2'

build-image:
	docker build --progress=plain -t $(IMAGE_REPOSITORY)/fat-controller/mc-ytt-bridge:$(IMAGE_VERSION) .

push-image: build-image
	docker push $(IMAGE_REPOSITORY)/fat-controller/mc-ytt-bridge:$(IMAGE_VERSION)

run-image:
	docker run --rm -p 8080:8080 $(IMAGE_REPOSITORY)/mc-ytt-bridge:$(IMAGE_VERSION)

prune-images:
	docker image prune --force

prune-docker:
	docker system prune --force

clean:
	-rm -f mc-ytt-bridge/mc-ytt-bridge
