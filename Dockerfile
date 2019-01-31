FROM golang:alpine as builder

ADD . /usr/src/sriov-network-device-plugin

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy
RUN apk add --update --virtual build-dependencies build-base linux-headers && \
    cd /usr/src/sriov-network-device-plugin && \
    make clean && \
    make build

FROM alpine
COPY --from=builder /usr/src/sriov-network-device-plugin/build/sriovdp /usr/bin/
WORKDIR /

LABEL io.k8s.display-name="SRIOV Network Device Plugin"

ADD ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
