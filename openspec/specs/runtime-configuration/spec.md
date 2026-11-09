# runtime-configuration Specification

## Purpose
Запускать бота с явными настройками маршрутизации и постоянного хранения, применяя совместимые изменения базы без потери исходных данных.

## Requirements

### Requirement: Provider-specific Yandex proxy
Приложение SHALL принимать `TBOT_YANDEX_PROXY_URL` и использовать явно заданный прокси только для запросов Яндекс Музыки.

#### Scenario: Dedicated proxy is configured
- **WHEN** указан корректный URL отдельного прокси
- **THEN** Яндекс Музыка использует его, а Telegram и остальные провайдеры сохраняют собственную маршрутизацию

#### Scenario: Proxy cannot be used
- **WHEN** прокси задан некорректно либо недоступен при запросе
- **THEN** некорректная настройка отклоняется при запуске, а ошибка запроса не вызывает скрытый прямой повтор в обход явного прокси

### Requirement: Persistent SQLite path
Бот SHALL использовать `TBOT_SQLITE_DATA_SOURCE` для пути SQLite, а Compose проекта SHALL подключать каталог базы с хоста для сохранения данных при пересоздании контейнера.

#### Scenario: Recreate the service
- **WHEN** контейнер пересоздан с тем же подключённым каталогом базы
- **THEN** ранее сохранённые ссылки и история доступны новому контейнеру

### Requirement: Automatic compatible initialization
Бот SHALL создавать нужные таблицы при запуске и однократно переносить старые ссылки, оставляя исходную таблицу `links`.

#### Scenario: Existing legacy database
- **WHEN** открыта база старого формата
- **THEN** корректные старые ссылки переносятся в кеш, исходная таблица сохраняется, а неизвестные даты не выдумываются

### Requirement: Observable cache outcomes without secret dumps
Логи SHALL отличать попадание в кеш, промах и ошибку, а обычный промах SHALL не выводиться как ошибка SQL; настройки с токенами и секретами SHALL не печататься при запуске.

#### Scenario: Cache miss
- **WHEN** готовая запись для ссылки отсутствует
- **THEN** лог показывает `cache=miss`, а не `sql: no rows in result set`

#### Scenario: Telegram response failure
- **WHEN** отправка ответа Telegram завершается ошибкой
- **THEN** ошибка записывается с ID чата и без открытого токена бота
