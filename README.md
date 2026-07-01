# Realty Parser

Сервис для автоматического сбора объявлений об аренде квартир с ЦИАН. Периодически обходит страницы поиска по заданным фильтрам, извлекает данные о каждом объявлении и сохраняет их в PostgreSQL и Kafka.

## Что делает сервис

- Обходит страницы поиска ЦИАН по настроенным фильтрам (цена, площадь, количество комнат)
- Для каждого нового объявления извлекает полную информацию: цена, площадь, этаж, метро, условия аренды и т.д.
- Рассчитывает скоринговый балл объявления для удобства сортировки
- Сохраняет данные в PostgreSQL (таблица `flats_history`)
- Публикует каждое новое объявление в топик Kafka (`realty.flats`) в формате JSON
- Уже виденные объявления пропускает — повторных записей нет

## Конфигурация

Все настройки задаются в `config.yaml`:

```yaml
search:
  min_price: 50000       # минимальная цена аренды
  max_price: 85000       # максимальная цена
  min_area: 38           # минимальная площадь
  rooms: [2, 3, 4]       # типы комнатности (2=1к, 3=2к, 4=3к)
  pages_limit: 1         # максимум страниц за один обход
  overall_limit: 20      # максимум объявлений за один обход

worker:
  period_seconds: 10            # интервал между обходами
  request_timeout_ms: 5000      # таймаут HTTP-запроса
  sleep_before_request_ms: 3000 # пауза перед каждым запросом к объявлению
  max_retries: 5                # количество повторных попыток при ошибке
  cookie_header: ""             # cookie сессии браузера (обновлять при блокировке)

database:
  dsn: "postgres://realty_parser:password@realty-postgres:5432/realty_parser?sslmode=disable"

kafka:
  brokers:
    - "realty-kafka:9092"
  topic: "realty.flats"

logging:
  file_path: "/var/log/realty-parser/app.log"
  level: "debug"   # debug / info / warn / error

metrics:
  port: 9095
```

## Метрики и здоровье

После запуска доступны:

```
GET http://localhost:9095/healthz   # health check
GET http://localhost:9095/metrics   # Prometheus метрики
```

---

## Запуск в Docker

Сервис запускается в контейнере в рамках общей Docker-сети `realty-net`. PostgreSQL и Kafka должны быть уже подняты — см. инструкцию в `dev-tips/`.

```bash
bash server_setup.sh
```

Собирает Docker-образ и запускает контейнер `realty-parser`. Логи пишутся в `/tmp/realty-parser-logs/` на хосте.

```
# Логи:
docker logs -f realty-parser

# Остановить:
docker stop realty-parser
```
