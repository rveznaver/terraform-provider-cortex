HOSTNAME=registry.terraform.io
NAMESPACE=inuits
NAME=cortex
BINARY=terraform-provider-${NAME}
VERSION=0.6.0
OS_ARCH=darwin_amd64

# Release build: strip symbols and debug info, trim paths, inject version/commit (smaller binaries)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
RELEASE_FLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -trimpath

# Terraform registry recommended OS/arch: https://developer.hashicorp.com/terraform/registry/providers/os-arch
RELEASE_PLATFORMS=darwin/amd64 darwin/arm64 linux/386 linux/amd64 linux/arm linux/arm64 windows/386 windows/amd64 freebsd/386 freebsd/amd64

default: install

.PHONY: build
build:
	go build -o ${BINARY}

.PHONY: release
release:
	@mkdir -p ./bin
	@for p in $(RELEASE_PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build $(RELEASE_FLAGS) -o ./bin/$(BINARY)_$(VERSION)_$${os}_$${arch} .; \
	done

.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

.PHONY: test
test:
	go test ./... -timeout=30s -parallel=4 -count=1 -v

.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -timeout 120m -v -count=1

.PHONY: clean
clean:
	rm -f examples/terraform.tfstate
	rm -f examples/terraform.tfstate.backup

dev.tfrc:
	echo 'provider_installation {' >> dev.tfrc
	echo '  dev_overrides {' >> dev.tfrc
	echo '    "form3tech-oss/cortex" = "$(CURDIR)"' >> dev.tfrc
	echo '  }' >> dev.tfrc
	echo '  direct {}' >> dev.tfrc
	echo '}' >> dev.tfrc

.PHONY: cortex-up
cortex-up:
	docker compose up -d

.PHONY: cortex-down
cortex-down:
	docker compose down
