# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace

# install packr
RUN go get -u github.com/gobuffalo/packr/packr

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY scripts/ scripts/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on packr build -a -o manager main.go
# Install insecure_key
COPY docker/insecure_key /root/insecure_key
RUN chmod 600 /root/insecure_key

FROM alpine:3.12
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /root/insecure_key /root/insecure_key

ENTRYPOINT ["/manager"]
