FROM fedora:32

RUN dnf install -y --best --refresh  --setopt=tsflags=nodocs git wget make golang cpio && dnf clean all && rm -rf /var/cache/dnf/*

ENV GOPATH=/go

WORKDIR $GOPATH/src/github.com/intel/userspace-cni-network-plugin

RUN git clone --depth 1 --single-branch --branch master https://github.com/intel/userspace-cni-network-plugin $GOPATH/src/github.com/intel/userspace-cni-network-plugin

RUN make install && make

CMD cp userspace/userspace /cni
