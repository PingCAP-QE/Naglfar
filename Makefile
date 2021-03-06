
# Image URL to use all building/pushing image targets
IMG ?= naglfar:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

CONTROLLER_GEN=$(GOBIN)/controller-gen
PACKR=$(GOBIN)/packr

PACKAGES := go list ./...| grep -vE 'vendor' | grep 'github.com/PingCAP-QE/Naglfar/'
PACKAGE_DIRECTORIES := $(PACKAGES) | sed 's|github.com/PingCAP-QE/Naglfar/||'

all: manager kubectl-naglfar

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: mod generate fmt vet
	go build -o bin/manager main.go

pack: mod generate fmt vet
	$(PACKR) build -o bin/manager main.go

kubectl-naglfar: mod generate fmt vet
	go build -o bin/naglfar cmd/kubectl-naglfar/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: docker-build
	kustomize build config/default | kubectl apply -f -
	kubectl apply -f relationship/machine-testresource.yaml

destroy: manifests
	kustomize build config/default | kubectl delete -f -

upgrade: deploy
	kubectl rollout restart deployment/naglfar-controller-manager -n naglfar-system

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

format: vet fmt

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

mod:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	@git diff --exit-code -- go.sum go.mod

# Generate code
generate: manifests
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build daemon image
docker-build-daemon:
	cp ./cmd/daemon/deploy.go ./docker/daemon/deploy.go
	docker build ./docker/daemon -t pingcapqe/upload-daemon
	rm ./docker/daemon/deploy.go

# Push daemon image
docker-push-daemon:
	docker push pingcapqe/upload-daemon

# Build the docker image
docker-build: manifests generate
	DOCKER_BUILDKIT=1 docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

groupimports: install-goimports
	goimports -w -l -local github.com/PingCAP-QE/Naglfar $$($(PACKAGE_DIRECTORIES))

install-goimports:
ifeq (,$(shell which goimports))
	@echo "installing goimports"
	go get golang.org/x/tools/cmd/goimports
endif

log:
	kubectl logs -f deployment/naglfar-controller-manager -n naglfar-system -c manager

gen-proto:
	protoc --go_out=plugins=grpc:. pkg/chaos/pb/chaos.proto

install-controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
endif

install-kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_TMP_DIR ;\
	wget https://github.com/kubernetes-sigs/kustomize/archive/kustomize/v3.8.8.tar.gz; \
	tar xvf v3.8.8.tar.gz; \
	cd kustomize-kustomize-v3.8.8/kustomize/; \
	go install; \
	rm -rf $$KUSTOMIZE_TMP_DIR ;\
	}
endif

install-protoc-gen:
ifeq (, $(shell which protoc-gen-go))
	@{ \
	set -e ;\
	PROTOC_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$PROTOC_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/golang/protobuf/protoc-gen-go@v1.2.0 ;\
	rm -rf $$PROTOC_GEN_TMP_DIR ;\
	}
endif


install-packr:
ifeq (, $(shell which packr))
	@{ \
	set -e ;\
	PACKER_TMP_DIR=$$(mktemp -d) ;\
	cd $$PACKER_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/gobuffalo/packr/packr@v0.2.5 ;\
	rm -rf $$PACKER_TMP_DIR ;\
	}
endif
