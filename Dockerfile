FROM golang:1.24 AS base

WORKDIR /usr/src/app

FROM base AS dev

FROM base AS builder

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod ./
# COPY go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o /usr/local/bin/app .

CMD ["app"]
