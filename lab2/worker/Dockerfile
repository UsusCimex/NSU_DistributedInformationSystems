# Worker/Dockerfile
# Этап сборки
FROM golang:1.20-alpine AS builder
WORKDIR /app
# Используем корневой контекст: копируем файлы модуля worker
COPY worker/go.mod worker/go.sum ./
# Копируем общую директорию common из корня
COPY common/ ../common/
# Копируем исходники сервиса worker (вся директория worker)
COPY worker/ .
RUN go mod download
RUN go build -o worker ./cmd/worker

# Финальный образ
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/worker .
CMD ["./worker"]
