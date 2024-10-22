# syntax=docker/dockerfile:1
FROM golang:1.23.2-alpine AS builder
RUN apk add --no-cache make gcc musl-dev linux-headers git ca-certificates
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go install .

FROM alpine:3.20
RUN apk add --no-cache curl
COPY --from=builder /go/bin/* /usr/local/bin/
EXPOSE 9090
ENTRYPOINT [ "metis-sequencer-exporter" ]
