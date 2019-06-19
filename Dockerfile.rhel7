FROM registry.svc.ci.openshift.org/ocp/builder:golang-1.10 AS builder

ADD . /usr/src/sriov-network-device-plugin

WORKDIR /usr/src/sriov-network-device-plugin

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy
RUN make clean && \
    make build

FROM registry.svc.ci.openshift.org/ocp/4.0:base
ENV INSTALL_PKGS "hwdata"
RUN yum install -y $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all
COPY --from=builder /usr/src/sriov-network-device-plugin/build/sriovdp /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="SRIOV Network Device Plugin"

ADD ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
