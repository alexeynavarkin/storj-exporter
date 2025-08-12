FROM golang:1.24.3-alpine3.20 AS builder

WORKDIR /build
COPY . .
RUN go build -o storj-exporter ./cmd/exporter/main.go

FROM alpine:3.20
WORKDIR /storj-exporter
COPY --from=builder /build/storj-exporter .
CMD [ "./storj-exporter" ]
