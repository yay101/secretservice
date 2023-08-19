# syntax=docker/dockerfile:1
##Build
FROM golang:1.19-alpine AS build
WORKDIR /src
RUN apk add --update gcc musl-dev git && git clone https://github.com/yay101/secretservice && cd /src/secretservice && go build -ldflags="-s -w" -o /src/secretservice
##Deploy
FROM alpine:latest
WORKDIR /app
COPY --from=build /src/secretservice /app/secretservice
CMD ["/app/secretservice"]
EXPOSE 3000
