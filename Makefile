# Go params
GO_BIN_FOLDER=$(GOPATH)/bin
GO_PREFIX=GOOS=${GOOS} GOARM=${GOARM} GOARCH=${GOARCH} PATH=${PATH}:$(GO_BIN_FOLDER)
GOCMD=$(GO_PREFIX) GO111MODULE=on go
GOCMD_NO_MOD=$(GO_PREFIX) GO111MODULE=off go

GOGET=$(GOCMD) get
GOBUILD=$(GOCMD) build -ldflags="-s -w" ${BUILD_EXTRA}
GOGENERATE=$(GOCMD_NO_MOD) generate

MOD=$(GOCMD) mod tidy
MOD_RESTORE=$(GOGET) -v -d ./... && $(GOCMD) mod vendor
GOVERALLS=$(GO_BIN_FOLDER)/goveralls

PLUGINS_LOCATION=$(GOPATH)/src/go-home.io/x/providers/
BIN_FOLDER=${CURDIR}/bin
BIN_NAME=$(BIN_FOLDER)/go-home
PLUGINS_BINS=$(BIN_FOLDER)/plugins

METALINER=GO111MODULE=off PATH=${PATH}:$(BIN_FOLDER) $(BIN_FOLDER)/gometalinter --sort=linter --config=${CURDIR}/.gometalinter.json

.PHONY: utilities-build utilities-ci utilities build-server build-plugins build run-server run-worker test-local test lint-local vendor-cleanup run-only-server dep-shared-update generate-local

define build_plugins_task =
	set -e
	rm -rf $(PLUGINS_BINS)
	cd $(PLUGINS_LOCATION)
	for plugin_type in *; do
		if [ -d "$${plugin_type}" ]; then
			for plugin in $${plugin_type}/*; do
				if [ -d "$${plugin}" ]; then
					cd $${plugin}
					echo "======================================="
					echo "Building $${plugin}"
					echo "======================================="
					$(GOBUILD) -buildmode=plugin -o $(PLUGINS_BINS)/$${plugin}.so .
					cd $(PLUGINS_LOCATION)
				fi;
			done;
		fi;
	done;
endef

define validate_dependencies =
	set -e
	echo "======================================="
	echo "Validating server"
	echo "======================================="
	cd plugins
	$(MOD)
	cd ..
	$(MOD)
	cd $(PLUGINS_LOCATION)
	for plugin_type in *; do
		if [ -d "$${plugin_type}" ]; then
			for plugin in $${plugin_type}/*; do
				if [ -d "$${plugin}" ]; then
					cd $${plugin}
					echo "======================================="
					echo "Validating $${plugin}"
					echo "======================================="
					$(MOD)
					cd $(PLUGINS_LOCATION)
				fi;
			done;
		fi;
	done;
endef

define update_shared_package_dependency =
	set -e
	echo "======================================="
	echo "Updating server"
	echo "======================================="
	cd plugins
	$(MOD)
	cd ..
	$(GOGET) go-home.io/x/server/plugins@master
	$(MOD)
	cd $(PLUGINS_LOCATION)
	for plugin_type in *; do
		if [ -d "$${plugin_type}" ]; then
			for plugin in $${plugin_type}/*; do
				if [ -d "$${plugin}" ]; then
					cd $${plugin}
					echo "======================================="
					echo "Updating $${plugin}"
					echo "======================================="
					$(GOGET) go-home.io/x/server/plugins@master
					$(MOD)
					cd $(PLUGINS_LOCATION)
				fi;
			done;
		fi;
	done;
endef

define restore_dependencies =
	set -e
	echo "======================================="
	echo "Validating server"
	echo "======================================="
	$(MOD_RESTORE)
	cd plugins
	$(MOD_RESTORE)
	cd $(PLUGINS_LOCATION)
	for plugin_type in *; do
		if [ -d "$${plugin_type}" ]; then
			for plugin in $${plugin_type}/*; do
				if [ -d "$${plugin}" ]; then
					cd $${plugin}
					echo "======================================="
					echo "Validating $${plugin}"
					echo "======================================="
					$(MOD_RESTORE)
					cd $(PLUGINS_LOCATION)
				fi;
			done;
		fi;
	done;
endef

define lint_all =
	set -e
	echo "======================================="
	echo "Linting server"
	echo "======================================="
    for fld in $$($(GOCMD) list ./...); do
        cd $${GOPATH}/src/$${fld}
        cwd=$$(pwd)
        if [ "$${cwd}" != "$(CURDIR)" ]; then
            echo $${cwd}
            $(METALINER) --enable=megacheck .
        fi;
	done;

	cd ${CURDIR}/plugins
	for fld in $$($(GOCMD) list ./...); do
		cd $${GOPATH}/src/$${fld}
		pwd
		$(METALINER) --enable=megacheck .
	done;

	cd $(PLUGINS_LOCATION)
	for plugin_type in *; do
		if [ -d "$${plugin_type}" ]; then
			for plugin in $${plugin_type}/*; do
				if [ -d "$${plugin}" ]; then
					echo "======================================="
					echo "Linting $${plugin}"
					echo "======================================="
					cd $${plugin}
					$(METALINER) --enable=megacheck_provider ./...
					cd $(PLUGINS_LOCATION)
				fi;
			done;
		fi;
    done;
endef

define lint_cleanup =
	set -e
	echo "======================================="
	echo "Cleaning up vendoring"
	echo "======================================="
	cd $(CURDIR)
	rm -rf vendor
	cd plugins
	rm -rf vendor
	cd $(PLUGINS_LOCATION)
	for plugin_type in *; do
		if [ -d "$${plugin_type}" ]; then
			for plugin in $${plugin_type}/*; do
				if [ -d "$${plugin}" ]; then
					cd $${plugin}
					rm -rf vendor
					cd $(PLUGINS_LOCATION)
				fi;
			done;
		fi;
	done;
endef

define go_generate =
	set -e
	echo "======================================="
	echo "Autogenerating"
	echo "======================================="
	cd $(CURDIR)
	$(GOGENERATE) -v ./...
	cd plugins
	$(GOGENERATE) -v ./...
	cd $(PLUGINS_LOCATION)
	for plugin_type in *; do
		if [ -d "$${plugin_type}" ]; then
			for plugin in $${plugin_type}/*; do
				if [ -d "$${plugin}" ]; then
					cd $${plugin}
					$(GOGENERATE) -v ./...
					cd $(PLUGINS_LOCATION)
				fi;
			done;
		fi;
	done;
endef

utilities-build:
	$(GOGET) github.com/alvaroloes/enumer
	$(GOGET) github.com/rakyll/statik

utilities-ci:
	curl -L https://git.io/vp6lP | sh
	$(GOGET) github.com/mattn/goveralls

utilities: utilities-build utilities-ci

build-server:
	$(GOBUILD) -ldflags "-X go-home.io/x/server/utils.Version=${VERSION} \
		-X go-home.io/x/server/utils.Arch=${GOARCH}" -o $(BIN_NAME)

build: build-plugins build-server

run-only-server:
	$(BIN_NAME) -c provider:fs -c location:${CURDIR}/configs -p ${CURDIR}/bin/plugins

run-server: build run-only-server

run-only-worker:
	$(BIN_NAME) -c provider:fs -c location:${CURDIR}/configs -p ${CURDIR}/bin/plugins -w

run-worker: build run-only-worker

test:
	@set -e
	$(GOCMD) test -failfast --covermode=count -coverprofile=$(BIN_FOLDER)/cover.out.tmp ./... ./plugins/...
	@cat $(BIN_FOLDER)/cover.out.tmp | grep -v "fake_" | grep -v "_enumer" | \
	    grep -v "mocks" | grep -v "public" | grep -v "statik" | \
	    grep -v "cmd/" | grep -v "errors.go" > $(BIN_FOLDER)/cover.out
	@rm -f $(BIN_FOLDER)/cover.out.tmp

test-local: test
	$(GOCMD) tool cover --html=$(BIN_FOLDER)/cover.out

git: vendor-cleanup dep-ensure generate lint-local test-local

build-rpi-cache-docker:
	docker build -t go-home-cahe -f Dockerfile.rpi.cache .

.ONESHELL:
SHELL = /bin/sh
build-plugins:
	$(build_plugins_task)

.ONESHELL:
SHELL = /bin/sh
dep:
	$(restore_dependencies)

.ONESHELL:
SHELL = /bin/sh
lint:
	$(lint_all)

lint-local: dep lint vendor-cleanup

.ONESHELL:
SHELL = /bin/sh
vendor-cleanup:
	$(lint_cleanup)

.ONESHELL:
SHELL = /bin/sh
dep-ensure:
	set -e
	$(validate_dependencies)
	cd ${CURDIR}
	$(GOCMD) run cmd/mod/mod.go ${CURDIR} $(PLUGINS_LOCATION)

.ONESHELL:
SHELL = /bin/sh
dep-shared-update:
	set -e
	$(update_shared_package_dependency)
	cd ${CURDIR}
	@$(MAKE) dep-ensure

.ONESHELL:
SHELL = /bin/sh
generate:
	$(go_generate)

generate-local: dep generate vendor-cleanup