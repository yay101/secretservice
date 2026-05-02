FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o secretservice

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

RUN addgroup -S secretservice && adduser -S secretservice -G secretservice

WORKDIR /app

COPY --from=builder /build/secretservice .

RUN mkdir -p /data && chown secretservice:secretservice /data

USER secretservice

ENV SS_DOMAIN=secretservice.au
ENV SS_PATH=/data
ENV SS_PROXY=false
ENV SS_PORT=80
ENV SS_SSL=443
ENV SS_CHUNK=1048576

EXPOSE 80 443

ENTRYPOINT ["./secretservice"]
