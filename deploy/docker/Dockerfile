# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/gim cmd/gim/main.go

# Runtime stage
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bin/gim /usr/local/bin/gim
COPY --from=builder /app/configs /etc/gim/configs
EXPOSE 8080 8081
ENTRYPOINT ["gim"]
