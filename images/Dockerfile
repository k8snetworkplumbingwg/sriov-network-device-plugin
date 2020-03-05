FROM golang:alpine as builder

COPY . /usr/src/sriov-network-device-plugin
ADD https://downloads.sourceforge.net/project/e1000/ddptool%20stable/ddptool-1.0.0.0/ddptool-1.0.0.0.tar.gz /tmp/ddptool/

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy
RUN env
RUN apk add --no-cache --virtual build-dependencies build-base linux-headers
WORKDIR  /usr/src/sriov-network-device-plugin
RUN make clean && \
    make build
WORKDIR /tmp/ddptool
RUN tar zxvf ddptool-1.0.0.0.tar.gz && \
        make
FROM alpine:3.7
RUN apk add --no-cache hwdata-pci
COPY --from=builder /usr/src/sriov-network-device-plugin/build/sriovdp /usr/bin/
COPY --from=builder /tmp/ddptool/ddptool /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="SRIOV Network Device Plugin"

COPY ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
