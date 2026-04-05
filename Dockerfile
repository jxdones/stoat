FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o stoat .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates libgcc

WORKDIR /app
COPY --from=builder /build/stoat /app/stoat

ENTRYPOINT ["/app/stoat"]
