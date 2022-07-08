FROM golang:1.17.3-alpine3.14

RUN apk --update add curl git openssh
WORKDIR /app

ENV GO111MODULE=on GOOS=linux CGO_ENABLED=0