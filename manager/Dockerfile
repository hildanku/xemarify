FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /out/xemarify-manager ./cmd/main.go

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app

COPY --from=builder /out/xemarify-manager /usr/local/bin/xemarify-manager

EXPOSE 8089

CMD ["xemarify-manager"]
