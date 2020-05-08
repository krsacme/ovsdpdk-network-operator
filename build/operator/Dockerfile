FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/ovsdpdk-network-operator

# install operator binary
COPY _output/bin/ovsdpdk-network-operator ${OPERATOR}

COPY bindata /bindata

CMD ["/usr/bin/ovsdpdk-network-operator"]
