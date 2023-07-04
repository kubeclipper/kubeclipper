FROM --platform=${BUILDPLATFORM}  golang:1.19-alpine3.18 as build
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ENV GOPROXY=https://goproxy.cn
WORKDIR /kubeclipper
COPY . /kubeclipper
RUN KUBE_BUILD_PLATFORMS="${TARGETPLATFORM}" KUBE_ALL_WITH_PREFIX="true" make build

FROM alpine:3.18
ARG TARGETOS
ARG TARGETARCH
WORKDIR /root
COPY --from=build /kubeclipper/dist/${TARGETOS}_${TARGETARCH}/kubeclipper-server .
ENTRYPOINT  ["./kubeclipper-server","serve"]
