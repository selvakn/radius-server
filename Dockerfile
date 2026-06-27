FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o radius-server ./cmd/server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/radius-server .
EXPOSE 1812/udp
EXPOSE 1813/udp
EXPOSE 8080/tcp
ENTRYPOINT ["/app/radius-server"]
CMD ["--config", "/etc/radius-server/config.yaml"]
