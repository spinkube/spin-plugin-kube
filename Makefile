COMMIT = $(shell git rev-parse --short HEAD)

.PHONY: build
build:
	go build -ldflags "-X github.com/spinkube/spin-plugin-kube/pkg/cmd.Version=git-${COMMIT}" -o bin/spin-kube .

.PHONY: install
install:
	spin pluginify --install
