ARG BUILADH_IMG
# fedora based official builadh image
FROM ${BUILADH_IMG}

#
# Setup Go develop environments
RUN mkdir -p /go
ENV GOPATH=/go
RUN dnf install --setopt=install_weak_deps=False --best -y golang-1.13.5 \
    #
    # Verify git, process tools, lsb-release (common in install instructions for CLIs) installed
    && dnf install --setopt=install_weak_deps=False --best -y git iproute procps redhat-lsb \
    #
    # Install Go tools w/module support
    && mkdir -p /tmp/gotools \
    && cd /tmp/gotools \
    && GO111MODULE=on go get -v golang.org/x/tools/gopls@latest 2>&1 \
    && GO111MODULE=on go get -v \
        honnef.co/go/tools/...@latest \
        golang.org/x/tools/cmd/gorename@latest \
        golang.org/x/tools/cmd/goimports@latest \
        golang.org/x/tools/cmd/guru@latest \
        golang.org/x/lint/golint@latest \
        github.com/mdempsky/gocode@latest \
        github.com/cweill/gotests/...@latest \
        github.com/haya14busa/goplay/cmd/goplay@latest \
        github.com/sqs/goreturns@latest \
        github.com/josharian/impl@latest \
        github.com/davidrjenni/reftools/cmd/fillstruct@latest \
        github.com/uudashr/gopkgs/cmd/gopkgs@latest  \
        github.com/ramya-rao-a/go-outline@latest  \
        github.com/acroca/go-symbols@latest  \
        github.com/godoctor/godoctor@latest  \
        github.com/rogpeppe/godef@latest  \
        github.com/zmb3/gogetdoc@latest \
        github.com/fatih/gomodifytags@latest  \
        github.com/mgechev/revive@latest  \
        github.com/go-delve/delve/cmd/dlv@latest 2>&1 \
    #
    # Install Go tools w/o module support
    && go get -v github.com/alecthomas/gometalinter 2>&1 \
    #
    # Install gocode-gomod
    && go get -x -d github.com/stamblerre/gocode 2>&1 \
    && go build -o gocode-gomod github.com/stamblerre/gocode \
    && mv gocode-gomod $GOPATH/bin/ \
    #
    # Install golangci-lint
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin 2>&1 \
    #
    # Clean up
    && rm -rf  /go/src /tmp/gotools

ENV GO111MODULE=on
ENV PATH=${GOPATH}/bin:${PATH}
