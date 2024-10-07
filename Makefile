MAKE_REL_PATH:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

GOLANG ?= go
GO_ENV_ARM64 ?= GOOS=linux GOARCH=arm64
GO_BUILD_FLAGS ?= -ldflags="-s -w"
GO_LAMBDA_TAGS ?= -tags "lambda.norpc"
GO_ARM64_BUILD ?= ${GO_ENV_ARM64} ${GOLANG} build ${GO_BUILD_FLAGS}

.PHONY: clean
clean:
	rm -rf ./bin

GO_BUILD_FILES := $(shell find . -type f -not -path "./.git/*" -not -path "./bin/*" -not -name "*_test.go")

bin/%/bootstrap: ${GO_BUILD_FILES}
	cd cmd/$(patsubst bin/%/bootstrap,%,$@) && ${GO_ARM64_BUILD} ${GO_LAMBDA_TAGS} -o ${MAKE_REL_PATH}/$@

.PHONY: build
build: bin/s3-to-datadog-push/bootstrap
build: bin/google-workspace-poll/bootstrap
	ln -sfn ${MAKE_REL_PATH}/credentials.json ${MAKE_REL_PATH}/bin/google-workspace-poll/credentials.json

.PHONY: deploy-production
deploy-production: build
	sam deploy
