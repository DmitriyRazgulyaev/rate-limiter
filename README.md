# rate-limiter

Простой rate limiter на Go, реализованный по алгоритму **Token Bucket**. Может использоваться как HTTP middleware для ограничения количества запросов от одного IP-адреса.

## Как это работает

Каждому клиенту (по IP) выделяется «ведро» токенов:

- при инициализации у клиента есть `MaxTokens` токенов;
- каждый запрос тратит один токен;
- если токенов не осталось — запрос отклоняется с кодом `429 Too Many Requests`;
- токены постепенно восполняются со временем, с интервалом `RefillInterval`, до тех пор пока их количество не достигнет `MaxTokens`.

Хранение состояния клиента (сколько токенов осталось, когда было последнее пополнение) вынесено за интерфейс `Repository`, что позволяет использовать любое хранилище (in-memory, Redis и т.д.) для трекинга состояния.

## Структура проекта

```
.
├── dto/            # структуры данных (например, User — состояние клиента)
├── repository/     # реализация хранилища состояния клиентов
├── test/           # тестовый клиент
├── limiter.go      # основная логика лимитера (Token Bucket)
├── middleware.go    # HTTP middleware на основе лимитера
├── go.mod
└── go.sum
```

## Основные компоненты

### `Limiter`

```go
type Limiter struct {
    repo            Repository
    TTL             time.Duration
    MaxTokens       int
    RefillInterval  int
}

func NewRateLimiter(repository Repository, TTL time.Duration, MaxTokens int, RefillInterval int) *Limiter
```

Создаёт новый лимитер с указанным хранилищем, временем жизни записи клиента (`TTL`), максимальным количеством токенов (`MaxTokens`) и интервалом пополнения токенов в секундах (`RefillInterval`).

Метод `Check(ctx, ip)` возвращает `true`, если запрос разрешён (и списывает токен), либо `false`, если лимит исчерпан.

### `Repository`

Интерфейс хранилища, которое должно уметь получать и сохранять состояние клиента по IP:

```go
type Repository interface {
    GetUser(ctx context.Context, ip string, TTL time.Duration, MaxTokens int) (dto.User, error)
    SetUser(ctx context.Context, ip string, tokensRemaining int, lastRefill time.Time, TTL time.Duration) (dto.User, error)
}
```

### `Middleware`

HTTP middleware, который перед каждым запросом проверяет IP клиента через `Limiter.Check` и возвращает `429 Too Many Requests`, если лимит превышен:

```go
func Middleware(limiter RateLimiter) func(http.Handler) http.Handler
```

## Пример использования

```go
package main

import (
    "net/http"
    "time"

    rate_limiter "rate-limiter"
    "rate-limiter/repository"
)

func main() {
    repo := repository.New() // ваша реализация Repository

    limiter := rate_limiter.NewRateLimiter(
        repo,
        10*time.Minute, // TTL записи клиента
        5,              // максимум 5 токенов
        10,             // пополнение раз в 10 секунд
    )

    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("ok"))
    })

    handler := rate_limiter.Middleware(limiter)(mux)

    http.ListenAndServe(":8080", handler)
}
```

## Установка

```bash
go get github.com/DmitriyRazgulyaev/rate-limiter
```

## Статус проекта

Проект находится в разработке — часть обработки ошибок в `middleware.go` помечена как `TODO` (проверка кодов ошибок).
