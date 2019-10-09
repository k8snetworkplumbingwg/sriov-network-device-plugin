FROM golang:alpine as builder

ADD . /usr/src/sriov-network-device-plugin
ADD https://downloads.sourceforge.net/project/e1000/ddptool%20stable/ddptool-1.0.0.0/ddptool-1.0.0.0.tar.gz /tmp/ddptool/

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy
RUN apk add --update --virtual build-dependencies build-base linux-headers && \
    cd /usr/src/sriov-network-device-plugin && \
    make clean && \
    make build && \
    cd /tmp/ddptool && tar zxvf ddptool-1.0.0.0.tar.gz && make

FROM alpine
RUN apk add hwdata-pci
COPY --from=builder /usr/src/sriov-network-device-plugin/build/sriovdp /usr/bin/
COPY --from=builder /tmp/ddptool/ddptool /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="SRIOV Network Device Plugin"

ADD ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
