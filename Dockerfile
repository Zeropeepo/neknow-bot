# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api/main.go

# Stage 2: Run
FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache curl ca-certificates

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]