# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install:
	kubectl apply -f config/crd/bases/

# Uninstall CRDs from a cluster
uninstall:
	kubectl delete -f config/crd/bases/

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy:
	kubectl apply -f config/rbac/deploy_admin_rbac.yaml
	#kubectl apply -f config/rbac/deploy_rbac.yaml
	kubectl apply -f config/manager/deploy_manager.yaml

undeploy:
	kubectl delete -f config/manager/deploy_manager.yaml
	#kubectl delete -f config/rbac/deploy_rbac.yaml
	kubectl delete -f config/rbac/deploy_admin_rbac.yaml
	

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build:
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .


# Custom targets for Template operator
.PHONY: test-gen test-crd test-verify test-lint test-unit

# Test if zz_generated.deepcopy.go file is generated
test-gen: save-sha-gen generate compare-sha-gen

# Test if crd yaml files are generated
test-crd: save-sha-crd manifests compare-sha-crd

# Verify if go.sum is valid
test-verify: save-sha-mod verify compare-sha-mod

# Test code lint
test-lint:
	#golangci-lint run ./... -v -E gofmt --timeout 1h0m0s
	golint ./...

# Unit test
test-unit:
	go test -v ./controllers/...

save-sha-gen:
	$(eval GENSHA=$(shell sha512sum api/v1/zz_generated.deepcopy.go))

compare-sha-gen:
	$(eval GENSHA_AFTER=$(shell sha512sum api/v1/zz_generated.deepcopy.go))
	@if [ "${GENSHA_AFTER}" = "${GENSHA}" ]; then echo "zz_generated.deepcopy.go is not changed"; else echo "zz_generated.deepcopy.go file is changed"; exit 1; fi

save-sha-crd:
	$(eval CRDSHA1=$(shell sha512sum config/crd/bases/tmax.io_catalogserviceclaims.yaml))
	$(eval CRDSHA2=$(shell sha512sum config/crd/bases/tmax.io_clustertemplates.yaml))
	$(eval CRDSHA3=$(shell sha512sum config/crd/bases/tmax.io_templateinstances.yaml))
	$(eval CRDSHA4=$(shell sha512sum config/crd/bases/tmax.io_templates.yaml))

compare-sha-crd:
	$(eval CRDSHA1_AFTER=$(shell sha512sum config/crd/bases/tmax.io_catalogserviceclaims.yaml))
	$(eval CRDSHA2_AFTER=$(shell sha512sum config/crd/bases/tmax.io_clustertemplates.yaml))
	$(eval CRDSHA3_AFTER=$(shell sha512sum config/crd/bases/tmax.io_templateinstances.yaml))
	$(eval CRDSHA4_AFTER=$(shell sha512sum config/crd/bases/tmax.io_templates.yaml))
	@if [ "${CRDSHA1_AFTER}" = "${CRDSHA1}" ]; then echo "tmax.io_catalogserviceclaims.yaml is not changed"; else echo "tmax.io_catalogserviceclaims.yaml file is changed"; exit 1; fi
	@if [ "${CRDSHA2_AFTER}" = "${CRDSHA2}" ]; then echo "tmax.io_clustertemplates.yaml is not changed"; else echo "tmax.io_clustertemplates.yaml file is changed"; exit 1; fi
	@if [ "${CRDSHA3_AFTER}" = "${CRDSHA3}" ]; then echo "tmax.io_templateinstances.yaml is not changed"; else echo "tmax.io_templateinstances.yaml file is changed"; exit 1; fi
	@if [ "${CRDSHA4_AFTER}" = "${CRDSHA4}" ]; then echo "tmax.io_templates.yaml is not changed"; else echo "tmax.io_templates.yaml file is changed"; exit 1; fi

save-sha-mod:
	$(eval MODSHA=$(shell sha512sum go.mod))
	$(eval SUMSHA=$(shell sha512sum go.sum))

verify:
	go mod verify

compare-sha-mod:
	$(eval MODSHA_AFTER=$(shell sha512sum go.mod))
	$(eval SUMSHA_AFTER=$(shell sha512sum go.sum))
	@if [ "${MODSHA_AFTER}" = "${MODSHA}" ]; then echo "go.mod is not changed"; else echo "go.mod file is changed"; exit 1; fi
	@if [ "${SUMSHA_AFTER}" = "${SUMSHA}" ]; then echo "go.sum is not changed"; else echo "go.sum file is changed"; exit 1; fi