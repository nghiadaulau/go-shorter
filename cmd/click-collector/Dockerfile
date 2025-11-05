FROM golang:1.24-alpine AS build
RUN apk --no-cache add git
RUN adduser -u 1001 app -D
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG GIT_COMMIT=dev
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w -X 'main.Version=${GIT_COMMIT}'" -o /out/exec ./cmd/click-collector

FROM alpine:3.20
WORKDIR /app
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /out/exec /app/exec
USER 1001
ENTRYPOINT ["/app/exec"]


