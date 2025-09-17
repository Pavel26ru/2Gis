# Код‑ревью: Прототип сервиса бронирования (2GIS)

## Краткое резюме
- **Объем**: Один эндпоинт для создания бронирования. Слои: обработчики (`internal/api`), сервисы (`internal/services`), репозитории (`internal/repositories`), DTO (`internal/dto`).
- **Главные риски**: утечка бизнес‑логики в репозиторий, не потокобезопасное in‑memory хранилище (риск двойного бронирования), слабая модель ошибок и маппинг в HTTP‑коды, использование DTO как доменной модели, смешение ответственности валидации и проверки доступности, отсутствие абстракций времени/ID, низкая тестопригодность.
- **Неотложные правки**: перенести бизнес‑правила в сервис; сделать репозиторий чистым портом персистентности; добавить мьютекс в in‑memory реализацию; ввести доменные ошибки и маппинг в HTTP; усилить валидацию и нормализацию входа; разделить домен и DTO; подключить middleware таймаутов и recovery.

## Архитектура assessment (Clean Architecture / слоистость)
- **API layer (controllers/handlers)**: `CreateOrderHandler` парсит/валидирует запрос и вызывает сервис. Валидация минимальная и находится в хендлере; маппинг ошибок в HTTP грубый (всегда 500). Маршрутизация на chi — ок.
- **Service layer (use‑cases)**: `ordersService.Create` лишь проксирует в репозиторий и возвращает тот же DTO. Отсутствуют ключевые правила кейса: проверка доступности, существования номера, корректности диапазона дат, идемпотентность, инварианты.
- **Repository layer (adapters)**: `orderMemoryRepository` содержит бизнес‑правила (`isRoomAvailable` и `isKnownRoom`) и состояние. Это нарушает разделение: адаптеры не должны владеть доменными правилами, только предоставлять операции хранения/чтения.
- **DTOs vs Домен**: Есть только `dto.Order` и `dto.Room`, используемые во всех слоях. Нет доменной модели и value‑объектов; повышается связность с транспортом и затрудняется эволюция.
- **Interfaces**: `OrdersRepository` содержит лишь `Create`. Он не покрывает запросы, нужные сервису для проверки доступности, из‑за чего правила утекают в репозиторий. `OrdersService` возвращает DTO, а не доменную сущность.
- **Composition**: `cmd/api/main.go` связывает memory‑репозиторий с хардкодом номеров; для демо ок, но нет абстракции конфигурации. В случае ошибки сервера — panic.

## Конкурентность и корректность
- **Data race / двойное бронирование**: `orderMemoryRepository` читает/пишет срез `orders` без синхронизации; пара «проверка доступности» + `append` не атомарна. При параллельных запросах возможны два успешных бронирования. Нужен `sync.RWMutex` или единая критическая секция для «проверка+вставка». В БД — транзакции и ограничения (уникальные/исключающие).
- **Сравнение времени**: Проверка перекрытия `order.From.Before(to) && order.To.After(from)` корректна для полуоткрытых интервалов, но сервис должен явно определить включительность границ и нормализовать до «дат» при необходимости.

## Обработка ошибок и API‑контракты
- **Handler**: Все ошибки маппятся в 500; ошибки валидации — 400, конфликты — 409, не найдено — 404, некорректные данные — 422 и т.п. Нужны структурированные ответы об ошибках.
- **Типы ошибок**: Ввести устойчивые доменные ошибки (`ErrInvalidDateRange`, `ErrUnknownRoom`, `ErrRoomNotAvailable`) и маппить их в хендлере.
- **Логирование**: Структурированный логгер; избегать panic в `main`; завершаться с корректным кодом возврата.

## Валидация и инварианты
- Сейчас проверяется только `from > to`. Отсутствует: непустые hotel/room/email, запрет `from == to`, ограничение максимальной длительности, нормализация таймзон, базовая проверка email, запрет бронирования «в прошлое» (по требованию бизнеса).

## Расширяемость и интерфейсы
- Репозитории должны предоставлять запросы, нужные сервису, например: `GetRoomBy(hotelID, roomTypeID)`, `ListOrdersByRoomAndRange(...)`, либо специализированный `IsRoomAvailable(...)`, реализованный на уровне хранения, но оркестрируемый правилом сервиса.
- Разделить интерфейсы чтения и записи при необходимости (CQRS‑дружественно).

## Рекомендуемая структура проекта
Текущая:
- `cmd/api/main.go`
- `internal/api/*`
- `internal/services/*`
- `internal/repositories/*`
- `internal/dto/*`
- `internal/interfaces/*`

Предлагаемая (пошагово, Go‑френдли, чистые слои):
- `cmd/api/` — composition root
- `cmd/grpc/` — gRPC сервер (альтернативный транспорт)
- `internal/app/http/handlers/` — HTTP‑хендлеры, DTO запрос/ответ, маппинг ошибок
- `internal/app/grpc/handlers/` — gRPC хендлеры, protobuf контракты
- `internal/app/http/middleware/` — HTTP middleware (auth, logging, metrics, recovery)
- `internal/app/grpc/middleware/` — gRPC interceptors (auth, logging, metrics, recovery)
- `internal/domain/booking/` — сущности (`Order`, `RoomType`, `Room`), value‑объекты (`DateRange`, `Email`), доменные ошибки
- `internal/usecase/booking/` — сервисы/кейсы (`CreateBooking`), порты (интерфейсы) их зависимостей
- `internal/adapter/repository/memory/` — in‑memory реализация с мьютексом
- `internal/adapter/repository/postgres/` — будущая БД‑реализация
- `internal/adapter/notification/email/` — будущий email‑адаптер
- `internal/adapter/notification/sms/` — будущий SMS‑адаптер
- `internal/platform/config/` — конфигурация
- `internal/platform/log/` — логгер
- `internal/platform/metrics/` — Prometheus метрики, регистрация
- `internal/platform/tracing/` — OpenTelemetry трассировка
- `internal/platform/clock/` — абстракция времени для тестов
- `internal/platform/id/` — генерация ID
- `internal/platform/health/` — health checks
- `api/proto/` — protobuf схемы для gRPC
- `deployments/docker/` — Dockerfile, docker-compose
- `deployments/k8s/` — Kubernetes манифесты
- `deployments/monitoring/` — Prometheus, Grafana конфиги
- `scripts/` — миграции БД, утилиты
- `pkg/` — переиспользуемые библиотеки (если появятся)

## Concrete refactor plan (phased)
1) Quick safety and correctness
- Add domain errors and HTTP mapping in handler.
- Move validation to service; handler only binds and delegates.
- Add `sync.RWMutex` to in-memory repo; guard read/write and ensure check+append is atomic.
- Improve `main` error handling and add `middleware.Recoverer`, `middleware.Timeout`.

2) Introduce domain and ports
- Define `domain/booking` with `Order`, `Room`, `DateRange`, domain errors.
- Change `OrdersService` to accept domain model or a command struct and return domain entity.
- Expand repository port: `FindRoom`, `ListOrdersByRoomAndRange` or `CreateIfAvailable` with atomic semantics for memory store; DB impl will use transactions.

3) Observability and config
- Use structured logger (`zap` or stdlog wrapper). Add request ID middleware.
- Config via env (`PORT`, etc.).

4) Testing
- Unit tests for service rules (date overlap, known room, errors).
- Concurrency test ensuring no double-booking in memory repo.
- HTTP handler tests mapping errors to statuses.

5) Future features hooks
- Email notification port; async dispatcher interface.
- Discount/Promo port; pricing service placeholder.

## Suggested interfaces and signatures
- Service (use‑case):
```go
// CreateBookingCommand is an input model detached from transport
type CreateBookingCommand struct {
	HotelID    string
	RoomTypeID string
	UserEmail  string
	From       time.Time
	To         time.Time
}

type BookingService interface {
	Create(ctx context.Context, cmd CreateBookingCommand) (Order, error)
}
```
- Repository (ports):
```go
type OrdersRepository interface {
	// Atomically persist if still available; for DB impl use tx/constraints
	Create(ctx context.Context, order Order) error
	ListByRoomAndRange(ctx context.Context, hotelID, roomTypeID string, from, to time.Time) ([]Order, error)
}

type RoomsRepository interface {
	Get(ctx context.Context, hotelID, roomTypeID string) (Room, error)
}
```
- Domain errors:
```go
var (
	ErrInvalidDateRange  = errors.New("invalid date range")
	ErrUnknownRoom       = errors.New("unknown room")
	ErrRoomNotAvailable  = errors.New("room not available")
)
```

## HTTP error mapping policy
- `ErrInvalidDateRange` → 400 Bad Request
- `ErrUnknownRoom` → 404 Not Found (or 400 if we avoid room enumeration leakage)
- `ErrRoomNotAvailable` → 409 Conflict
- Other errors → 500 Internal Server Error

## Concurrency guard for memory repo (sketch)
```go
type orderMemoryRepository struct {
	mu     sync.RWMutex
	orders []Order
	rooms  map[key]Room
}

func (r *orderMemoryRepository) Create(ctx context.Context, o Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// validate existence using r.rooms
	// check overlap against r.orders for same room
	// append
	return nil
}
```

## Handler improvements (sketch)
- Bind and call service; map domain errors; return JSON with `id` (when introduced) and echo of data.
- Add `middleware.Recoverer`, `middleware.Timeout(3s)`.

## Interview plan (step-by-step)
1) Walkthrough of current architecture and data flow
- Show `main.go` wiring → handler → service → repo. Discuss where rules reside and why that's problematic.

2) Identify key issues
- Business logic in repo; race conditions; weak error handling; lack of domain separation; DTO misuse; minimal validation; no observability/config.

3) Discuss clean layering and proposed structure
- Present domain/use-case/adapters/platform split; explain benefits for future features (DB, email, discounts, multi-room bookings).

4) Live refactor (short)
- Move validation to service; add domain errors; update handler mapping.
- Add mutex to repo; make check+append atomic.

5) Concurrency deep-dive
- Explain overlap logic; atomicity; DB-level constraints vs app-level locks; idempotency keys.

6) Evolution scenarios
- Adding real DB: outline repository interfaces; transactions; indexes; exclusion constraints for overlapping bookings.
- Email confirmation: define notifier port and async worker.
- Discounts/promocodes: pricing service port; compute final price in service with extensible policies.
- Multi-room booking: introduce aggregate `Booking` with multiple `BookedRoom` items; transactional creation.

7) Testing strategy
- Unit tests for service invariants; table-driven tests for overlaps; concurrency tests; handler contract tests.

8) Operational concerns
- Config, logging, tracing, health checks, graceful shutdown, timeouts.

## Concrete TODOs (shortlist)
- Introduce domain errors and map to HTTP.
- Move validation into service; strengthen validation rules.
- Add `sync.RWMutex` to memory repo; guard atomic create.
- Split DTOs vs domain; create basic domain models.
- Expand repository interface to support queries needed by rules.
- Improve `main` error handling; add recovery/timeout middlewares.
- Add unit and concurrency tests.

## Notes on DB-scale design
- Start with PostgreSQL; tables: `hotels`, `room_types`, `rooms`, `users`, `orders`, `payments` with FKs and indexes.
- Enforce non-overlapping bookings per room: exclusion constraint on `orders` with gist index over `tsrange(from, to)`.
- Use read replicas, caching of static refs (rooms, hotels), and partition `orders` by date if needed.

## gRPC и мониторинг (дополнения)

### gRPC сервис
- **Структура**: `cmd/grpc/` для отдельного gRPC сервера, `internal/app/grpc/handlers/` для обработчиков
- **Контракты**: `api/proto/booking.proto` с сервисом `BookingService` и методами `CreateBooking`, `GetBooking`, `ListBookings`
- **Middleware**: interceptors для auth, logging, metrics, recovery, rate limiting
- **Преимущества**: типизированные контракты, потоковая передача, бинарный протокол, мультиязычность

### Мониторинг и метрики
- **Prometheus**: `internal/platform/metrics/` для регистрации метрик, `/metrics` endpoint
- **Метрики**: RED (Requests, Errors, Duration), бизнес-метрики (bookings_created_total, booking_conflicts_total)
- **Grafana**: дашборды в `deployments/monitoring/grafana/`, алерты на error rate и latency
- **Трассировка**: OpenTelemetry в `internal/platform/tracing/`, спаны для всех операций
- **Логирование**: структурированные логи с request_id, централизованный сбор в ELK/Loki

### Деплой и инфраструктура
- **Docker**: `deployments/docker/` с multi-stage builds, health checks
- **Kubernetes**: `deployments/k8s/` с ConfigMaps, Secrets, HPA
- **Мониторинг**: `deployments/monitoring/` с Prometheus, Grafana, AlertManager конфигами
