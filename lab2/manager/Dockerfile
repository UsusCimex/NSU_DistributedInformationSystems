# Этап сборки
FROM golang:1.24-alpine AS builder

# Создаем рабочую директорию
WORKDIR /app

# Копируем общий модуль
COPY common /app/common

# Копируем файлы модуля manager
COPY manager /app/manager

# Устанавливаем зависимости и собираем приложение
RUN cd /app/manager && go mod download
RUN cd /app/manager && go build -o manager cmd/manager/main.go

# Финальный образ
FROM alpine:latest

# Устанавливаем зависимости для Alpine
RUN apk --no-cache add ca-certificates tzdata

# Копируем исполняемый файл из этапа сборки
COPY --from=builder /app/manager/manager /usr/local/bin/manager

# Задаем порт для HTTP сервера
EXPOSE 8080

# Задаем пользователя для безопасности
RUN adduser -D appuser
USER appuser

# Запускаем приложение
CMD ["manager"]
