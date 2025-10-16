# syntax=docker/dockerfile:1.7-labs

FROM golang:1.24 as dev
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

FROM golang:1.24 as builder
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/shortlink-api ./cmd/shortlink-api && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/redirector ./cmd/redirector

FROM alpine:3.20 as certs
RUN apk add --no-cache ca-certificates && update-ca-certificates

FROM gcr.io/distroless/static:nonroot as runtime
WORKDIR /app
COPY --from=builder /out/shortlink-api /app/shortlink-api
COPY --from=builder /out/redirector /app/redirector
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/app/shortlink-api"]
