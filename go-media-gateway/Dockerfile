FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/gateway ./cmd/main.go

FROM alpine:3.20
RUN apk add --no-cache ca-certificates su-exec \
    && mkdir -p /var/lib/stackmesh/blobs
COPY --from=build /out/gateway /usr/local/bin/gateway
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod 755 /docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["/docker-entrypoint.sh"]
