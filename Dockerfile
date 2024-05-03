# Dockerfile
FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build -o rate-limiter .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=build /app/rate-limiter .
CMD ["./rate-limiter"]
