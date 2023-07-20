FROM golang:1.19-alpine3.18 as build
ENV GOPROXY=https://goproxy.cn
WORKDIR /kubeclipper
COPY . /kubeclipper
RUN apk add make bash which && make build

FROM alpine:3.18
WORKDIR /root
COPY --from=build /kubeclipper/dist/kubeclipper-server .
ENTRYPOINT  ["./kubeclipper-server","serve"]