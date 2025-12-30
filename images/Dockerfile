FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
ENV HTTP_PROXY=$http_proxy
ENV HTTPS_PROXY=$https_proxy

WORKDIR /usr/src/sriov-network-device-plugin
COPY . .

RUN apk add --no-cache --virtual build-dependencies build-base
RUN make clean && make build

FROM --platform=$BUILDPLATFORM golang:1.20-alpine3.16 AS ddp-builder

ENV HTTP_PROXY=$http_proxy
ENV HTTPS_PROXY=$https_proxy

ADD images/ddptool-1.0.1.12.tar.gz /tmp/ddptool/
RUN apk add --no-cache --virtual build-dependencies build-base linux-headers

WORKDIR /tmp/ddptool
RUN make

FROM alpine:3
RUN apk add --no-cache hwdata-pci
COPY --from=builder /usr/src/sriov-network-device-plugin/build/sriovdp /usr/bin/
COPY --from=ddp-builder /tmp/ddptool/ddptool /usr/bin/
COPY ./images/entrypoint.sh /

WORKDIR /
LABEL io.k8s.display-name="SRIOV Network Device Plugin"
ENTRYPOINT ["/entrypoint.sh"]
