FROM centos:centos7

ADD . /usr/src/sriov-network-device-plugin

ENV INSTALL_PKGS "git golang make"
RUN yum install -y $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    cd /usr/src/sriov-network-device-plugin && \
    make clean && \
    make build && \
    yum autoremove -y $INSTALL_PKGS && \
    yum clean all && \
    rm -rf /tmp/*

WORKDIR /

LABEL io.k8s.display-name="SRIOV Network Device Plugin"

ADD ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
