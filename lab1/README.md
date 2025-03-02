# Распределенная информационная система по расшифровке MD5 хэша

## Описание проекта

Система представляет собой распределённое приложение для подбора исходных строк по их MD5-хэшам методом перебора *(bruteforce)*. Состоит из двух основных компонентов:
- Manager: координирует работу, распределяет задачи и обрабатывает запросы клиентов
- Worker: выполняет непосредственный перебор и поиск совпадений хэшей

## Проблемы текущей реализации

1. **Отсутствие персистентности данных**
   - Все данные хранятся только в оперативной памяти менеджера
   - При падении менеджера теряются все активные задачи и их результаты
   - Нет возможности восстановить состояние системы после сбоя

2. **Ненадежная коммуникация**
   - Прямое HTTP взаимодействие между компонентами без промежуточного слоя
   - При недоступности менеджера ответы от воркеров теряются
   - При недоступности воркера задача не может быть переназначена

3. **Отсутствие масштабируемости** *(UPD: в новой реализации у менеджера есть массив URL воркеров, с распределением по Round Robin)*
   - Система работает только с одним воркером
   - Нет механизма балансировки нагрузки
   - Отсутствует возможность горизонтального масштабирования

4. **Отсутствие отказоустойчивости**
   - Нет реплицирования данных
   - Единая точка отказа в виде менеджера
   - Отсутствует механизм перезапуска и восстановления компонентов

## Запуск проекта

```bash
docker compose up --build -d
```

## API Endpoints

### Manager Public API

#### POST /api/hash/crack
Отправляет запрос на расшифровку хэша.

Request:
```json
{
    "hash": "098f6bcd4621d373cade4e832627b4f6",
    "maxLength": 4
}
```

Response:
```json
{
    "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### GET /api/hash/status?requestId={requestId}
Получает статус расшифровки по requestId.

Response:
```json
{
    "status": "DONE|IN_PROGRESS|FAIL",
    "data": ["найденная_строка"]
}
```

### Manager Internal API

#### POST /internal/api/manager/hash/crack/request
Внутренний endpoint для получения результатов от worker'ов.

Request:
```json
{
    "hash": "098f6bcd4621d373cade4e832627b4f6",
    "result": "test",
    "partNumber": 1
}
```

### Worker API

#### POST /internal/api/worker/hash/crack/task
Endpoint для получения заданий на расшифровку от manager'а.

Request:
```json
{
    "hash": "098f6bcd4621d373cade4e832627b4f6",
    "maxLength": 4,
    "partNumber": 1,
    "partCount": 8
}
```

## Тестирование системы

Для тестирования используйте утилиту из [директории test](test):

```bash
cd test
go run main.go
```

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
