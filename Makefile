SHELL := /bin/bash
VERSION := $(shell cat VERSION)

release:
	goreleaser release --rm-dist
	docker buildx build --platform="linux/amd64" -t lunemec/eve-accountant:$(VERSION) .
	docker push lunemec/eve-accountant:$(VERSION)
