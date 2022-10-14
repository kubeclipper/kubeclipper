# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: generate deps
generate: deps
	./hack/generate_types.sh

deps:
    ifeq (, $(shell which client-gen))
		go install k8s.io/code-generator/cmd/{client-gen,lister-gen,informer-gen,deepcopy-gen,defaulter-gen,conversion-gen,openapi-gen}@v0.22.3
		CLIENT_GEN=$(GOBIN)/client-gen
    else
		CLIENT_GEN=$(shell which client-gen)
    endif

.PHONY: build build-server build-agent build-proxy build-cli openapi
build: build-server build-agent build-proxy build-cli

build-server:
	KUBE_VERBOSE=2 bash hack/make-rules/build.sh cmd/kubeclipper-server

build-agent:
	KUBE_VERBOSE=2 bash hack/make-rules/build.sh cmd/kubeclipper-agent

build-proxy:
	KUBE_VERBOSE=2 bash hack/make-rules/build.sh cmd/kubeclipper-proxy

build-cli:
	KUBE_VERBOSE=2 bash hack/make-rules/build.sh cmd/kcctl

build-e2e:
	go test -c -ldflags "-s -w" -o dist/e2e.test ./test/e2e

openapi:
	go run ./tools/doc-gen/main.go

.PHONY:test
test:
	go test -race ./pkg/... -coverprofile=coverage.out -covermode=atomic

.PHONY: format-deps checkfmt fmt goimports vet lint
format-deps:
    ifeq (, $(shell which golangci-lint))
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0
    endif
    ifeq (, $(shell which goimports))
		go install golang.org/x/tools/cmd/goimports@latest
    endif

checkfmt: format-deps fmt goimports lint

fmt:
	gofmt -s -w ./pkg ./cmd ./tools ./test

goimports:
	@hack/update-goimports.sh

vet:
	go vet ./pkg/... ./cmd/...

lint:
	golangci-lint run --timeout 10m


.PHONY: cli cleancli cli-serve
cli: cleancli
	go run ./tools/kcctldocs-gen/main.go
	docker run -v $(shell pwd)/tools/kcctldocs-gen/generators/includes:/source -v $(shell pwd)/tools/kcctldocs-gen/generators/build:/build -v $(shell pwd)/tools/kcctldocs-gen/generators/:/manifest brianpursley/brodocs:latest

cleancli:
	rm -rf $(shell pwd)/tools/kcctldocs-gen/generators/includes
	rm -rf $(shell pwd)/tools/kcctldocs-gen/generators/build
	rm -rf $(shell pwd)/tools/kcctldocs-gen/generators/manifest.json

cli-serve:cli
	docker stop kcctldocs
	docker rm -f kcctldocs
	docker build $(shell pwd)/tools/kcctldocs-gen/generators -t kubeclipper/kcctl:latest -f $(shell pwd)/tools/kcctldocs-gen/generators/Dockerfile
	docker run -p 8080:80 --name kcctldocs kubeclipper/kcctl:latest
