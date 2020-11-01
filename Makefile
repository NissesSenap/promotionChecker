GH_VERSION ?= $(shell git describe --tags 2>/dev/null || git rev-parse --short HEAD)
DATE_FMT = +%Y-%m-%d
BUILD_DATE = $(shell date "$(DATE_FMT)")

bin/pc: $(BUILD_FILES)
	@go build -o bin/promotionChecker ./main.go
	#@go build -o "$@" ./main.go

bin/container:
	podman build --build-arg BUILD_DATE=$(BUILD_DATE) --build-arg VERSION=$(GH_VERSION) .

clean:
	rm -rf ./bin ./share
.PHONY: clean

test:
	go test ./...
.PHONY: test
