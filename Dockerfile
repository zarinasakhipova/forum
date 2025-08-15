#######################
# 1. Build stage
#######################
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Установим C‑зависимости
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ❗ включаем CGO
ENV CGO_ENABLED=1
RUN go build -o forum cmd/main.go

#######################
# 2. Final stage
#######################
FROM alpine:latest
WORKDIR /app

# ❗ Установим минимальную C‑библиотеку (нужна, если CGO=1)
RUN apk add --no-cache libc6-compat

COPY --from=builder /app/forum ./forum
COPY static ./static
COPY templates ./templates
COPY forum.db ./forum.db

EXPOSE 8080
CMD ["./forum"]
