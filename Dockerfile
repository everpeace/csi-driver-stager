# build args should be declared when using multi-stage build
ARG BUILADH_IMG

# -----------------------------------------------------------------------------
# step 1: build
FROM golang:1.13 as builder

# for go mod download
RUN apt-get update -q && apt-get install -qy --no-install-recommends ca-certificates  git

RUN mkdir -p /src
WORKDIR /src
COPY go.mod .
COPY go.sum .
ENV GO111MODULE=on
RUN go mod download

RUN go get -u github.com/golangci/golangci-lint/cmd/golangci-lint \
  && go get -u golang.org/x/tools/cmd/goimports

COPY . .

RUN make build

# -----------------------------------------------------------------------------
# step 2: exec
FROM ${BUILADH_IMG} as runtime
COPY --from=builder /src/dist/csi-driver-stager /bin/csi-driver-stager
ENTRYPOINT ["/bin/csi-driver-stager"]
