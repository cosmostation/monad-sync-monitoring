COMMIT := $(shell git log -1 --format='%H')

BUILD_FLAGS := -tags $(BUILD_TAGS) -ldflags $(ldflags) -trimpath
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)ll
OS := $(shell [ -f /etc/os-release ] && . /etc/os-release && echo $$ID | sed -E 's/(ubuntu|debian|centos|rhel|fedora|arch).*/linux/' || go env GOOS)
SUFFIX := $(shell echo $$PLATFORM | sed 's/\//-/' | sed 's/\///')

GO_MIN_VERSION := 1.22

# Function to check go version
check_go_version:
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed."; exit 1; }
	@go_version=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	required_version=$(GO_MIN_VERSION); \
	if [ "$$(printf '%s\n' "$$required_version" "$$go_version" | sort -V | head -n1)" != "$$required_version" ]; then \
		echo "Go version $$go_version is less than required $$required_version"; \
		exit 1; \
	else \
		echo "Go version $$go_version is sufficient."; \
	fi

.PHONY: all
all: build

go.sum: go.mod
		@echo "--> Ensure dependencies have not been modified"
		GO111MODULE=on go mod verify

.PHONY: build
build: check_go_version go.sum
		go build -o ./bin/monad-monitoring -mod=readonly ./src
