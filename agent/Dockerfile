FROM golang:1.23-alpine AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /out/xemarify-agent .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata sed
RUN mkdir -p /etc/xemarify-agent /var/lib/xemarify-agent/spool

COPY --from=builder /out/xemarify-agent /usr/local/bin/xemarify-agent
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["docker-entrypoint.sh"]