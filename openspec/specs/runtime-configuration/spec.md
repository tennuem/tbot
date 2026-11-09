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
Бот SHALL создавать актуальные таблицы кеша и истории при запуске без переноса из старой таблицы `links`; старые данные чата загружаются явной командой импорта истории.

#### Scenario: Existing legacy database
- **WHEN** открыта база старого формата
- **THEN** таблица `links` остаётся без изменений и не попадает в актуальный кеш автоматически

#### Scenario: New or current database
- **WHEN** приложение открывает новую базу или базу с актуальными таблицами
- **THEN** нужные таблицы создаются при отсутствии, существующие кеш и история сохраняются, а таблицы `links` и `cache_migrations` не создаются

### Requirement: Observable cache outcomes without secret dumps
Логи SHALL отличать попадание в кеш, промах и ошибку, а обычный промах SHALL не выводиться как ошибка SQL; настройки с токенами и секретами SHALL не печататься при запуске.

#### Scenario: Cache miss
- **WHEN** готовая запись для ссылки отсутствует
- **THEN** лог показывает `cache=miss`, а не `sql: no rows in result set`

#### Scenario: Telegram response failure
- **WHEN** отправка ответа Telegram завершается ошибкой
- **THEN** ошибка записывается с ID чата и без открытого токена бота

### Requirement: Explicit and environment proxy precedence
Клиент Яндекс Музыки SHALL использовать явный HTTP(S)-прокси независимо от `HTTP_PROXY`, `HTTPS_PROXY` и `NO_PROXY`; при пустом `TBOT_YANDEX_PROXY_URL` SHALL сохранять стандартный выбор прокси Go из окружения.

#### Scenario: Explicit proxy overrides exclusions
- **WHEN** задан отдельный прокси и `NO_PROXY=*`
- **THEN** запрос Яндекс Музыки всё равно идёт через отдельный прокси

#### Scenario: No dedicated proxy
- **WHEN** `TBOT_YANDEX_PROXY_URL` пуст
- **THEN** маршрут определяется `http.ProxyFromEnvironment`, а отсутствие отдельной настройки не гарантирует прямое соединение
