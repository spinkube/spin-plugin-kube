COMMIT = $(shell git rev-parse --short HEAD)

.PHONY: build
build:
	go build -ldflags "-X github.com/spinkube/spin-plugin-k8s/pkg/cmd.Version=git-${COMMIT}" -o bin/spin-plugin-k8s .

.PHONY: install
install:
	spin pluginify --install
