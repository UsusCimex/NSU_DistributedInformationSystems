# Manager/Dockerfile
# Этап сборки
FROM golang:1.20-alpine AS builder
WORKDIR /app
# Используем корневой контекст: копируем файлы модуля manager
COPY manager/go.mod manager/go.sum ./
# Копируем общую директорию common из корня
COPY common/ ../common/
# Копируем исходники сервиса manager (вся директория manager)
COPY manager/ .
RUN go mod download
RUN go build -o manager ./cmd/manager

# Финальный образ
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/manager .
CMD ["./manager"]
