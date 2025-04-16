# Этап сборки
FROM golang:1.24-alpine AS builder

# Создаем рабочую директорию
WORKDIR /app

# Копируем общий модуль
COPY common /app/common

# Копируем файлы модуля worker
COPY worker /app/worker

# Устанавливаем зависимости и собираем приложение
RUN cd /app/worker && go mod download
RUN cd /app/worker && go build -o worker cmd/worker/main.go

# Финальный образ
FROM alpine:latest

# Устанавливаем зависимости для Alpine
RUN apk --no-cache add ca-certificates tzdata

# Копируем исполняемый файл из этапа сборки
COPY --from=builder /app/worker/worker /usr/local/bin/worker

# Задаем пользователя для безопасности
RUN adduser -D appuser
USER appuser

# Запускаем приложение
CMD ["worker"]
