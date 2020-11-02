GH_VERSION ?= $(shell git describe --tags 2>/dev/null || git rev-parse --short HEAD)
DATE_FMT = +%Y-%m-%d
BUILD_DATE = $(shell date "$(DATE_FMT)")
IMAGE_REPO = "quay.io/nissessenap/promotionchecker"
IMAGE_TEST_REPO = "quay.io/nissessenap/test-promotionchecker"

bin/pc: $(BUILD_FILES)
	@go build -o bin/promotionChecker ./main.go
	#@go build -o "$@" ./main.go

bin/container:
	podman build --build-arg BUILD_DATE=$(BUILD_DATE) --build-arg VERSION=$(GH_VERSION) . -t $(IMAGE_REPO):$(GH_VERSION)

bin/push:
	podman push $(IMAGE_REPO):$(GH_VERSION)

test/container:
	podman build --build-arg BUILD_DATE=$(BUILD_DATE) --build-arg VERSION=$(GH_VERSION) ./testServer -t $(IMAGE_TEST_REPO):$(GH_VERSION)

test/run-container:
	podman run -it -p 8081:8081 $(IMAGE_TEST_REPO):$(GH_VERSION)

test/push:
	podman push $(IMAGE_TEST_REPO):$(GH_VERSION)

test/helm:
	helm upgrade --install test-promotion testServer/test-promotion-checker

tekton/helm:
	helm upgrade --install tekton deploy/tekton-example

helm:
	helm upgrade --install promotion deploy/promoter

clean:
	rm -rf ./bin ./share
.PHONY: clean

test:
	go test ./...
.PHONY: test
