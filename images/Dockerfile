FROM golang:1.18-alpine3.16 as builder

COPY . /usr/src/sriov-network-device-plugin
ADD images/ddptool-1.0.1.12.tar.gz /tmp/ddptool/

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy
RUN apk add --no-cache --virtual build-dependencies build-base linux-headers git

WORKDIR /usr/src/sriov-network-device-plugin
RUN make clean && \
    make build

WORKDIR /tmp/ddptool
RUN make

FROM alpine:3
RUN apk add --no-cache hwdata-pci
COPY --from=builder /usr/src/sriov-network-device-plugin/build/sriovdp /usr/bin/
COPY --from=builder /tmp/ddptool/ddptool /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="SRIOV Network Device Plugin"

COPY ./images/entrypoint.sh /

RUN rm -rf /var/cache/apk/*

ENTRYPOINT ["/entrypoint.sh"]
