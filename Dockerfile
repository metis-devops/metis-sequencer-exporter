# syntax=docker/dockerfile:1
FROM golang:1.22.2 as builder
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go install .

FROM gcr.io/distroless/base-debian12
COPY --from=builder /go/bin/* /usr/local/bin/
EXPOSE 9090
ENTRYPOINT [ "metis-sequencer-exporter" ]
