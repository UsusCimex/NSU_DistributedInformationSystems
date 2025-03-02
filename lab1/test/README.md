### Тестирование системы

Доступные команды:

1. Получить MD5 хэш строки:
```bash
go run main.go -md5 "test"
```

2. Отправить задачу на расшифровку хэша:
```bash
go run main.go -crack <hash> [maxLength]
```
Пример:
```bash
go run main.go -crack 098f6bcd4621d373cade4e832627b4f6 4
```

3. Проверить статус расшифровки:
```bash
go run main.go -status <requestId>
```

### Пример использования

```bash
# Получаем хэш строки "test"
go run main.go -md5 "test"
# Вывод: 098f6bcd4621d373cade4e832627b4f6

# Отправляем задачу на расшифровку
go run main.go -crack 098f6bcd4621d373cade4e832627b4f6 4
# Вывод: RequestId: <some-uuid>

# Проверяем статус расшифровки
go run main.go -status <requestId>
```