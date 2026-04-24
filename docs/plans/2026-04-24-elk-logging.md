# ELK logging integration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Доставка структурных логов `orchv3` в ELK-стек для демо — TCP-sink в `steplog`, опциональный `io.MultiWriter` c stderr, `docker-compose` с Elasticsearch/Logstash/Kibana и автопровижинингом data view.

**Architecture:** Приложение продолжает писать JSON-события в stderr; параллельно они уходят в `TCPSink` (неблокирующий канал + фоновая горутина с reconnect) на Logstash `127.0.0.1:5000`. Logstash парсит `time → @timestamp`, ренеймит `type → level`, льёт в ES индекс `orchv3-YYYY.MM.dd`. Kibana на `5601` с преднастроенным data view. Всё включается одной env-переменной `LOGSTASH_ADDR`; пусто = фича выключена, приложение работает как сегодня.

**Tech Stack:** Go 1.24, stdlib (`net`, `bufio`, `encoding/json`, `sync`, `sync/atomic`, `time`, `io`), `github.com/joho/godotenv` (уже в проекте). Docker Compose, Elastic stack 8.x, curl для провижининга Kibana.

**Source spec:** `docs/specs/2026-04-24-elk-logging-design.md`

---

## Файловая структура

**Создаются:**
- `internal/steplog/tcp_sink.go` — тип `TCPSink`, конструктор, `Write`, `Close`, фоновая горутина с dial/reconnect/drain.
- `internal/steplog/tcp_sink_test.go` — юнит-тесты на delivery, non-blocking, overflow, reconnect, close-flush, drop-warnings.
- `cmd/orchv3/logger_setup.go` — хелпер `buildLogger(stderr, cfg) (steplog.Logger, io.Closer, error)`, выносим wiring из `main.go` чтобы тестировать отдельно.
- `cmd/orchv3/logger_setup_test.go` — тесты хелпера на включение/выключение sink, service-поле.
- `deploy/docker-compose.yml` — ES + Logstash + Kibana + kibana-setup.
- `deploy/logstash/pipeline/orchv3.conf` — input/filter/output.
- `deploy/kibana/saved-objects.ndjson` — data view `orchv3-*` (ручной экспорт дашборда добавляется в follow-up на этапе репетиции).
- `deploy/kibana/setup.sh` — ждёт Kibana, импортирует ndjson.
- `docs/elk-demo.md` — how-to: поднять стек, nc-smoke, URL дашборда, экспорт saved-objects.

**Модифицируются:**
- `internal/steplog/logger.go` — поле `Service` в `Event`, конструктор `NewWithService`, `Logger` получает `service`, передаётся в `Event`.
- `internal/steplog/logger_test.go` — case на `service`.
- `internal/config/config.go` — тип `LogstashConfig`, поле `Logstash` в `Config`, парсинг `LOGSTASH_*`, дефолты.
- `internal/config/config_test.go` — envKeys, defaults, custom, invalid buffer, invalid timeout.
- `cmd/orchv3/main.go` — вызов `buildLogger`, `defer closer.Close()`.
- `.env.example` — секция ELK.

---

## Task 1: Event.Service + NewWithService

Добавляем в `Event` поле `service` с `omitempty` и конструктор `NewWithService`. Существующий `New` продолжает работать и выставляет пустой `service` (опускается в JSON).

**Files:**
- Modify: `internal/steplog/logger.go`
- Modify: `internal/steplog/logger_test.go`

- [ ] **Step 1: Write the failing test**

Дописать в `internal/steplog/logger_test.go`:

```go
func TestNewWithServiceIncludesServiceField(t *testing.T) {
	var out bytes.Buffer

	NewWithService(&out, "orchv3-test").Infof("cli", "hello")

	event := decodeEvents(t, out.String())[0]
	if event.Service != "orchv3-test" {
		t.Fatalf("service = %q, want %q", event.Service, "orchv3-test")
	}
}

func TestNewOmitsServiceField(t *testing.T) {
	var out bytes.Buffer

	New(&out).Infof("cli", "hello")

	if strings.Contains(out.String(), `"service"`) {
		t.Fatalf("output contains service field: %s", out.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/steplog/ -run 'TestNewWithService|TestNewOmitsService' -v`
Expected: FAIL — `undefined: NewWithService` и `Event.Service` не существует.

- [ ] **Step 3: Add Service field and NewWithService constructor**

В `internal/steplog/logger.go`:

```go
type Event struct {
	Time    string `json:"time"`
	Service string `json:"service,omitempty"`
	Module  string `json:"module"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Logger struct {
	out     io.Writer
	service string
}

func New(out io.Writer) Logger {
	return NewWithService(out, "")
}

func NewWithService(out io.Writer, service string) Logger {
	if out == nil {
		out = io.Discard
	}
	return Logger{out: out, service: service}
}
```

И в `write`:

```go
func (logger Logger) write(module string, logType string, format string, args ...any) {
	event := Event{
		Time:    time.Now().UTC().Format(time.RFC3339Nano),
		Service: logger.service,
		Module:  normalizeModule(module),
		Type:    logType,
		Message: fmt.Sprintf(format, args...),
	}

	encoder := json.NewEncoder(logger.out)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(event)
}
```

- [ ] **Step 4: Run the whole steplog package tests**

Run: `go test ./internal/steplog/ -v`
Expected: все существующие тесты проходят + новые `TestNewWithServiceIncludesServiceField`, `TestNewOmitsServiceField` — PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/steplog/logger.go internal/steplog/logger_test.go
git commit -m "feat(steplog): add Service field and NewWithService"
```

---

## Task 2: LogstashConfig в `internal/config`

Добавляем секцию конфига. Дефолты: `Addr=""`, `BufferSize=1024`, `DialTimeout=2s`. Пустой `Addr` = фича выключена. Битый `BufferSize`/`DialTimeout` → ошибка `Load()`.

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

Добавить в `internal/config/config_test.go`:

1. В `envKeys` добавить три ключа:

```go
var envKeys = []string{
	// ...существующие
	"LOGSTASH_ADDR",
	"LOGSTASH_BUFFER_SIZE",
	"LOGSTASH_DIAL_TIMEOUT",
}
```

2. Расширить `TestLoadUsesDefaults` новым блоком в конце:

```go
	if cfg.Logstash.Addr != "" {
		t.Fatalf("Logstash.Addr = %q, want empty by default", cfg.Logstash.Addr)
	}
	if cfg.Logstash.BufferSize != defaultLogstashBufferSize {
		t.Fatalf("Logstash.BufferSize = %d, want %d", cfg.Logstash.BufferSize, defaultLogstashBufferSize)
	}
	if cfg.Logstash.DialTimeout != defaultLogstashDialTimeout {
		t.Fatalf("Logstash.DialTimeout = %v, want %v", cfg.Logstash.DialTimeout, defaultLogstashDialTimeout)
	}
```

3. Добавить новые тесты:

```go
func TestLoadReadsLogstashEnvironment(t *testing.T) {
	isolateEnv(t)
	t.Setenv("LOGSTASH_ADDR", "127.0.0.1:5000")
	t.Setenv("LOGSTASH_BUFFER_SIZE", "2048")
	t.Setenv("LOGSTASH_DIAL_TIMEOUT", "750ms")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Logstash.Addr != "127.0.0.1:5000" {
		t.Fatalf("Addr = %q", cfg.Logstash.Addr)
	}
	if cfg.Logstash.BufferSize != 2048 {
		t.Fatalf("BufferSize = %d", cfg.Logstash.BufferSize)
	}
	if cfg.Logstash.DialTimeout != 750*time.Millisecond {
		t.Fatalf("DialTimeout = %v", cfg.Logstash.DialTimeout)
	}
}

func TestLoadReturnsErrorForInvalidLogstashBufferSize(t *testing.T) {
	isolateEnv(t)
	t.Setenv("LOGSTASH_BUFFER_SIZE", "abc")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReturnsErrorForNonPositiveLogstashBufferSize(t *testing.T) {
	isolateEnv(t)
	t.Setenv("LOGSTASH_BUFFER_SIZE", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReturnsErrorForInvalidLogstashDialTimeout(t *testing.T) {
	isolateEnv(t)
	t.Setenv("LOGSTASH_DIAL_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}
```

Добавить импорт `"time"` вверху файла.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run 'Logstash' -v`
Expected: FAIL — `cfg.Logstash` не существует, `defaultLogstash*` не определены.

- [ ] **Step 3: Add LogstashConfig to config.go**

В `internal/config/config.go`:

1. Добавить импорт `"time"`.
2. Константы в общий `const`-блок:

```go
const (
	// ...существующие
	defaultLogstashBufferSize  = 1024
	defaultLogstashDialTimeout = 2 * time.Second
)
```

3. Типы:

```go
type Config struct {
	// ...существующие поля
	Logstash LogstashConfig
}

type LogstashConfig struct {
	Addr        string
	BufferSize  int
	DialTimeout time.Duration
}
```

4. В `Load()` перед `return` добавить:

```go
	logstashCfg, err := loadLogstashConfig()
	if err != nil {
		return Config{}, err
	}
```

И в возвращаемом `Config{...}` добавить поле `Logstash: logstashCfg,`.

5. Новая функция в конце файла:

```go
func loadLogstashConfig() (LogstashConfig, error) {
	bufferSize, err := intFromEnv("LOGSTASH_BUFFER_SIZE", defaultLogstashBufferSize)
	if err != nil {
		return LogstashConfig{}, err
	}
	if bufferSize < 1 {
		return LogstashConfig{}, fmt.Errorf("LOGSTASH_BUFFER_SIZE must be >= 1, got %d", bufferSize)
	}

	dialTimeout, err := durationFromEnv("LOGSTASH_DIAL_TIMEOUT", defaultLogstashDialTimeout)
	if err != nil {
		return LogstashConfig{}, err
	}

	return LogstashConfig{
		Addr:        trimmedStringFromEnv("LOGSTASH_ADDR", ""),
		BufferSize:  bufferSize,
		DialTimeout: dialTimeout,
	}, nil
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}

	return parsed, nil
}
```

- [ ] **Step 4: Run all config tests**

Run: `go test ./internal/config/ -v`
Expected: существующие + новые — все PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add LogstashConfig and LOGSTASH_* env parsing"
```

---

## Task 3: TCPSink — happy-path delivery

Минимальная реализация: канал, фоновая горутина, `net.Dial`, запись `bufio.Writer`. Пока без reconnect/overflow/warnings — только delivery.

**Files:**
- Create: `internal/steplog/tcp_sink.go`
- Create: `internal/steplog/tcp_sink_test.go`

- [ ] **Step 1: Write the failing test**

Создать `internal/steplog/tcp_sink_test.go`:

```go
package steplog

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func startTestListener(t *testing.T) (net.Listener, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener, listener.Addr().String()
}

func readLines(t *testing.T, listener net.Listener, want int, timeout time.Duration) []string {
	t.Helper()
	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	scanner := bufio.NewScanner(conn)
	lines := make([]string, 0, want)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) == want {
			break
		}
	}
	_ = conn.Close()
	return lines
}

func TestTCPSink_DeliversEvents(t *testing.T) {
	listener, addr := startTestListener(t)

	sink := NewTCPSink(addr, 16, 200*time.Millisecond, io.Discard)
	t.Cleanup(func() { _ = sink.Close() })

	for i := 0; i < 10; i++ {
		if _, err := sink.Write([]byte("event-" + string(rune('0'+i)) + "\n")); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	lines := readLines(t, listener, 10, 2*time.Second)
	if len(lines) != 10 {
		t.Fatalf("received %d lines, want 10: %v", len(lines), lines)
	}
	for i, line := range lines {
		want := "event-" + string(rune('0'+i))
		if !strings.HasPrefix(line, want) {
			t.Fatalf("line[%d] = %q, want prefix %q", i, line, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/steplog/ -run TestTCPSink_DeliversEvents -v`
Expected: FAIL — `NewTCPSink` не определён.

- [ ] **Step 3: Minimal implementation**

Создать `internal/steplog/tcp_sink.go`:

```go
package steplog

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPSink struct {
	addr        string
	dialTimeout time.Duration
	warnOut     io.Writer

	queue chan []byte

	dropped atomic.Uint64
	done    chan struct{}
	wg      sync.WaitGroup
}

func NewTCPSink(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer) *TCPSink {
	if bufferSize < 1 {
		bufferSize = 1
	}
	if warnOut == nil {
		warnOut = io.Discard
	}

	sink := &TCPSink{
		addr:        addr,
		dialTimeout: dialTimeout,
		warnOut:     warnOut,
		queue:       make(chan []byte, bufferSize),
		done:        make(chan struct{}),
	}

	sink.wg.Add(1)
	go sink.run()

	return sink
}

func (s *TCPSink) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	s.queue <- buf
	return len(p), nil
}

func (s *TCPSink) Close() error {
	close(s.done)
	s.wg.Wait()
	return nil
}

func (s *TCPSink) run() {
	defer s.wg.Done()
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := net.DialTimeout("tcp", s.addr, s.dialTimeout)
		if err != nil {
			select {
			case <-s.done:
				return
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}

		s.drain(conn)
	}
}

func (s *TCPSink) drain(conn net.Conn) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	for {
		select {
		case <-s.done:
			return
		case payload := <-s.queue:
			if _, err := writer.Write(payload); err != nil {
				return
			}
			if err := writer.Flush(); err != nil {
				return
			}
		}
	}
}

func (s *TCPSink) Dropped() uint64 {
	return s.dropped.Load()
}

// warnf writes a warning line to warnOut; errors ignored.
func (s *TCPSink) warnf(format string, args ...any) {
	_, _ = fmt.Fprintln(s.warnOut, fmt.Sprintf(format, args...))
}
```

- [ ] **Step 4: Run delivery test**

Run: `go test ./internal/steplog/ -run TestTCPSink_DeliversEvents -v`
Expected: PASS.

- [ ] **Step 5: Run whole package to make sure nothing broke**

Run: `go test ./internal/steplog/ -v`
Expected: все тесты — PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/steplog/tcp_sink.go internal/steplog/tcp_sink_test.go
git commit -m "feat(steplog): add TCPSink with happy-path delivery"
```

---

## Task 4: TCPSink — non-blocking Write и drop-counter

Сейчас `Write` блокирует при полном канале (goroutine не пишет, если dial не удался). Надо сделать неблокирующим: `select` с `default` → drop + инкремент счётчика.

**Files:**
- Modify: `internal/steplog/tcp_sink.go`
- Modify: `internal/steplog/tcp_sink_test.go`

- [ ] **Step 1: Write failing tests**

Дописать в `internal/steplog/tcp_sink_test.go`:

```go
func TestTCPSink_NonBlockingWrite(t *testing.T) {
	// No listener — DialTimeout fails, queue fills, further writes must not block.
	sink := NewTCPSink("127.0.0.1:1", 8, 50*time.Millisecond, io.Discard)
	t.Cleanup(func() { _ = sink.Close() })

	start := time.Now()
	for i := 0; i < 1000; i++ {
		if _, err := sink.Write([]byte("payload\n")); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Fatalf("1000 writes took %v, want < 200ms", elapsed)
	}
}

func TestTCPSink_DropsOnOverflow(t *testing.T) {
	// No listener, bufferSize=4, write 10 → 4 buffered, 6 dropped.
	sink := NewTCPSink("127.0.0.1:1", 4, 50*time.Millisecond, io.Discard)
	t.Cleanup(func() { _ = sink.Close() })

	for i := 0; i < 10; i++ {
		_, _ = sink.Write([]byte("x\n"))
	}

	// Give the goroutine a beat in case it raced a couple of items out of the queue.
	time.Sleep(10 * time.Millisecond)

	got := sink.Dropped()
	if got < 6 {
		t.Fatalf("Dropped() = %d, want >= 6", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/steplog/ -run 'TestTCPSink_NonBlockingWrite|TestTCPSink_DropsOnOverflow' -v`
Expected: FAIL — `TestTCPSink_NonBlockingWrite` виснет/таймаутит (канал полон, Write блокирует); `TestTCPSink_DropsOnOverflow` — `Dropped() = 0`.

- [ ] **Step 3: Make Write non-blocking**

Заменить метод `Write` в `internal/steplog/tcp_sink.go`:

```go
func (s *TCPSink) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case s.queue <- buf:
	default:
		s.dropped.Add(1)
	}
	return len(p), nil
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/steplog/ -run 'TestTCPSink_NonBlockingWrite|TestTCPSink_DropsOnOverflow' -v`
Expected: PASS.

- [ ] **Step 5: Run delivery test again to confirm no regression**

Run: `go test ./internal/steplog/ -run TestTCPSink_DeliversEvents -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/steplog/tcp_sink.go internal/steplog/tcp_sink_test.go
git commit -m "feat(steplog): make TCPSink.Write non-blocking with drop counter"
```

---

## Task 5: TCPSink — reconnect с экспоненциальным бэкофом

Заменить hardcoded `500ms` задержку на exponential backoff 1s → 30s. На успешном connect сбрасывать. Добавить тест на reconnect после рестарта сервера.

**Files:**
- Modify: `internal/steplog/tcp_sink.go`
- Modify: `internal/steplog/tcp_sink_test.go`

- [ ] **Step 1: Write failing test**

Дописать в `internal/steplog/tcp_sink_test.go`:

```go
func TestTCPSink_ReconnectsAfterServerRestart(t *testing.T) {
	listener1, addr := startTestListener(t)

	// Short reconnect baseline for the test.
	sink := newTCPSinkWithBackoff(addr, 16, 200*time.Millisecond, io.Discard, 50*time.Millisecond, 500*time.Millisecond)
	t.Cleanup(func() { _ = sink.Close() })

	_, _ = sink.Write([]byte("first\n"))
	lines := readLines(t, listener1, 1, 2*time.Second)
	if len(lines) != 1 || lines[0] != "first" {
		t.Fatalf("first batch = %v", lines)
	}

	_ = listener1.Close()

	// Bring up a new listener on the same port.
	listener2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("re-listen on %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = listener2.Close() })

	// Write until reconnect succeeds; goroutine backoff retries dial.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, _ = sink.Write([]byte("second\n"))
		time.Sleep(50 * time.Millisecond)
	}

	lines = readLines(t, listener2, 1, 3*time.Second)
	if len(lines) == 0 {
		t.Fatalf("no lines received after reconnect")
	}
	if lines[0] != "second" {
		t.Fatalf("post-reconnect line = %q, want %q", lines[0], "second")
	}
}
```

Здесь используется `newTCPSinkWithBackoff` — appendix-конструктор для тестов, который не светим в публичный API.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/steplog/ -run TestTCPSink_ReconnectsAfterServerRestart -v`
Expected: FAIL — `newTCPSinkWithBackoff` не определён.

- [ ] **Step 3: Add backoff machinery**

В `internal/steplog/tcp_sink.go`:

1. Добавить поля в `TCPSink`:

```go
type TCPSink struct {
	addr        string
	dialTimeout time.Duration
	warnOut     io.Writer
	backoffMin  time.Duration
	backoffMax  time.Duration

	queue chan []byte

	dropped atomic.Uint64
	done    chan struct{}
	wg      sync.WaitGroup
}
```

2. Константы:

```go
const (
	defaultBackoffMin = 1 * time.Second
	defaultBackoffMax = 30 * time.Second
)
```

3. Публичный конструктор теперь делегирует:

```go
func NewTCPSink(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer) *TCPSink {
	return newTCPSinkWithBackoff(addr, bufferSize, dialTimeout, warnOut, defaultBackoffMin, defaultBackoffMax)
}

func newTCPSinkWithBackoff(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer, backoffMin, backoffMax time.Duration) *TCPSink {
	if bufferSize < 1 {
		bufferSize = 1
	}
	if warnOut == nil {
		warnOut = io.Discard
	}

	sink := &TCPSink{
		addr:        addr,
		dialTimeout: dialTimeout,
		warnOut:     warnOut,
		backoffMin:  backoffMin,
		backoffMax:  backoffMax,
		queue:       make(chan []byte, bufferSize),
		done:        make(chan struct{}),
	}

	sink.wg.Add(1)
	go sink.run()

	return sink
}
```

4. Обновлённый `run`:

```go
func (s *TCPSink) run() {
	defer s.wg.Done()

	backoff := s.backoffMin
	firstFailureLogged := false

	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := net.DialTimeout("tcp", s.addr, s.dialTimeout)
		if err != nil {
			if !firstFailureLogged {
				s.warnf("steplog: logstash sink unavailable at %s, will retry: %v", s.addr, err)
				firstFailureLogged = true
			}
			select {
			case <-s.done:
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > s.backoffMax {
				backoff = s.backoffMax
			}
			continue
		}

		backoff = s.backoffMin
		firstFailureLogged = false
		s.drain(conn)
	}
}
```

- [ ] **Step 4: Run reconnect test**

Run: `go test ./internal/steplog/ -run TestTCPSink_ReconnectsAfterServerRestart -v -timeout 15s`
Expected: PASS.

- [ ] **Step 5: Run whole package**

Run: `go test ./internal/steplog/ -v -timeout 30s`
Expected: все тесты — PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/steplog/tcp_sink.go internal/steplog/tcp_sink_test.go
git commit -m "feat(steplog): reconnect with exponential backoff in TCPSink"
```

---

## Task 6: TCPSink.Close — flush pending + timeout

Сейчас `Close` просто закрывает `done`. Горутина прервёт drain и недоставленные события потеряются. Нужно: при Close горутина сливает оставшиеся payload-ы в текущее соединение с общим таймаутом 2s, затем закрывает conn.

**Files:**
- Modify: `internal/steplog/tcp_sink.go`
- Modify: `internal/steplog/tcp_sink_test.go`

- [ ] **Step 1: Write failing test**

Дописать в `internal/steplog/tcp_sink_test.go`:

```go
func TestTCPSink_CloseFlushesPending(t *testing.T) {
	listener, addr := startTestListener(t)

	sink := NewTCPSink(addr, 64, 200*time.Millisecond, io.Discard)

	for i := 0; i < 20; i++ {
		_, _ = sink.Write([]byte(fmt.Sprintf("evt-%02d\n", i)))
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	lines := readLines(t, listener, 20, 3*time.Second)
	if len(lines) != 20 {
		t.Fatalf("received %d lines, want 20: %v", len(lines), lines)
	}
	for i, line := range lines {
		want := fmt.Sprintf("evt-%02d", i)
		if line != want {
			t.Fatalf("line[%d] = %q, want %q", i, line, want)
		}
	}
}
```

Добавить импорт `"fmt"` в тестовом файле, если ещё нет.

- [ ] **Step 2: Run test to verify it may race/fail**

Run: `go test ./internal/steplog/ -run TestTCPSink_CloseFlushesPending -v -count=10`
Expected: FAIL/flaky — часть событий теряется потому что `drain` прерывается сразу при закрытии `done`.

- [ ] **Step 3: Add flush-on-close semantics**

Изменить метод `drain` и добавить вспомогательный флаш. В `internal/steplog/tcp_sink.go`:

```go
const closeFlushTimeout = 2 * time.Second

func (s *TCPSink) drain(conn net.Conn) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	for {
		select {
		case <-s.done:
			s.flushRemaining(writer)
			return
		case payload := <-s.queue:
			if _, err := writer.Write(payload); err != nil {
				return
			}
			if err := writer.Flush(); err != nil {
				return
			}
		}
	}
}

func (s *TCPSink) flushRemaining(writer *bufio.Writer) {
	deadline := time.Now().Add(closeFlushTimeout)
	for time.Now().Before(deadline) {
		select {
		case payload := <-s.queue:
			if _, err := writer.Write(payload); err != nil {
				return
			}
		default:
			_ = writer.Flush()
			return
		}
	}
	_ = writer.Flush()
}
```

- [ ] **Step 4: Run close test (stress with -count)**

Run: `go test ./internal/steplog/ -run TestTCPSink_CloseFlushesPending -v -count=20 -timeout 60s`
Expected: все 20 прогонов — PASS.

- [ ] **Step 5: Run whole package**

Run: `go test ./internal/steplog/ -v -timeout 30s`
Expected: все тесты — PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/steplog/tcp_sink.go internal/steplog/tcp_sink_test.go
git commit -m "feat(steplog): TCPSink.Close flushes pending queue with timeout"
```

---

## Task 7: TCPSink — periodic drop summary warning

Раз в N секунд, если `dropped` вырос с прошлого отчёта — одна строка в `warnOut`. Это помогает на демо заметить перегрузку стека. Для тестируемости выносим интервал в параметр.

**Files:**
- Modify: `internal/steplog/tcp_sink.go`
- Modify: `internal/steplog/tcp_sink_test.go`

- [ ] **Step 1: Write failing test**

Дописать в `internal/steplog/tcp_sink_test.go`:

```go
func TestTCPSink_PeriodicDropWarning(t *testing.T) {
	var warn bytes.Buffer
	sink := newTCPSinkWithBackoffAndWarnInterval(
		"127.0.0.1:1", 2, 50*time.Millisecond, &warn,
		50*time.Millisecond, 200*time.Millisecond, 100*time.Millisecond,
	)
	t.Cleanup(func() { _ = sink.Close() })

	for i := 0; i < 20; i++ {
		_, _ = sink.Write([]byte("x\n"))
	}

	// Wait long enough for at least one tick of the warning interval (100ms).
	time.Sleep(300 * time.Millisecond)

	output := warn.String()
	if !strings.Contains(output, "dropped") {
		t.Fatalf("warn output missing drop summary: %q", output)
	}
}
```

Добавить импорт `"bytes"` в тестовом файле, если ещё нет.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/steplog/ -run TestTCPSink_PeriodicDropWarning -v`
Expected: FAIL — `newTCPSinkWithBackoffAndWarnInterval` не определён.

- [ ] **Step 3: Add warning goroutine**

В `internal/steplog/tcp_sink.go`:

1. Константа:

```go
const defaultDropWarnInterval = 30 * time.Second
```

2. Новое поле в структуре:

```go
type TCPSink struct {
	// ...существующие
	warnInterval time.Duration
}
```

3. Публичный конструктор делегирует в новый ярус:

```go
func NewTCPSink(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer) *TCPSink {
	return newTCPSinkWithBackoffAndWarnInterval(
		addr, bufferSize, dialTimeout, warnOut,
		defaultBackoffMin, defaultBackoffMax, defaultDropWarnInterval,
	)
}

func newTCPSinkWithBackoff(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer, backoffMin, backoffMax time.Duration) *TCPSink {
	return newTCPSinkWithBackoffAndWarnInterval(
		addr, bufferSize, dialTimeout, warnOut,
		backoffMin, backoffMax, defaultDropWarnInterval,
	)
}

func newTCPSinkWithBackoffAndWarnInterval(
	addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer,
	backoffMin, backoffMax, warnInterval time.Duration,
) *TCPSink {
	if bufferSize < 1 {
		bufferSize = 1
	}
	if warnOut == nil {
		warnOut = io.Discard
	}

	sink := &TCPSink{
		addr:         addr,
		dialTimeout:  dialTimeout,
		warnOut:      warnOut,
		backoffMin:   backoffMin,
		backoffMax:   backoffMax,
		warnInterval: warnInterval,
		queue:        make(chan []byte, bufferSize),
		done:         make(chan struct{}),
	}

	sink.wg.Add(2)
	go sink.run()
	go sink.warnLoop()

	return sink
}
```

4. Горутина с отчётом:

```go
func (s *TCPSink) warnLoop() {
	defer s.wg.Done()
	if s.warnInterval <= 0 {
		return
	}
	ticker := time.NewTicker(s.warnInterval)
	defer ticker.Stop()

	var lastReported uint64
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			current := s.dropped.Load()
			if current > lastReported {
				s.warnf("steplog: dropped %d events due to sink overflow", current-lastReported)
				lastReported = current
			}
		}
	}
}
```

- [ ] **Step 4: Run warning test**

Run: `go test ./internal/steplog/ -run TestTCPSink_PeriodicDropWarning -v -timeout 15s`
Expected: PASS.

- [ ] **Step 5: Run whole package**

Run: `go test ./internal/steplog/ -v -timeout 30s`
Expected: все тесты — PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/steplog/tcp_sink.go internal/steplog/tcp_sink_test.go
git commit -m "feat(steplog): periodic drop-summary warnings from TCPSink"
```

---

## Task 8: `cmd/orchv3/logger_setup.go` — wire TCP sink на конфиг

Вытаскиваем сборку логгера в отдельный хелпер, чтобы тестировать. Возвращает `Logger`, `io.Closer` (может быть nil) и ошибку. Если `Addr` пуст — `Closer` nil, ничего не создаётся.

**Files:**
- Create: `cmd/orchv3/logger_setup.go`
- Create: `cmd/orchv3/logger_setup_test.go`

- [ ] **Step 1: Write failing test**

Создать `cmd/orchv3/logger_setup_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"

	"orchv3/internal/config"
)

type testEvent struct {
	Service string `json:"service"`
	Module  string `json:"module"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func TestBuildLogger_DisabledWhenAddrEmpty(t *testing.T) {
	var stderr bytes.Buffer

	logger, closer, err := buildLogger(&stderr, config.Config{
		AppName:  "orchv3",
		Logstash: config.LogstashConfig{Addr: ""},
	}, io.Discard)
	if err != nil {
		t.Fatalf("buildLogger: %v", err)
	}
	if closer != nil {
		t.Fatal("closer != nil when sink disabled")
	}
	logger.Infof("cli", "hello")

	var evt testEvent
	if err := json.Unmarshal(bytes.TrimSpace(stderr.Bytes()), &evt); err != nil {
		t.Fatalf("decode stderr: %v", err)
	}
	if evt.Service != "orchv3" {
		t.Fatalf("service = %q, want orchv3", evt.Service)
	}
}

func TestBuildLogger_FanoutsToSinkWhenEnabled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	var stderr bytes.Buffer
	logger, closer, err := buildLogger(&stderr, config.Config{
		AppName: "orchv3",
		Logstash: config.LogstashConfig{
			Addr:        listener.Addr().String(),
			BufferSize:  16,
			DialTimeout: 200 * time.Millisecond,
		},
	}, io.Discard)
	if err != nil {
		t.Fatalf("buildLogger: %v", err)
	}
	if closer == nil {
		t.Fatal("closer == nil when sink enabled")
	}
	t.Cleanup(func() { _ = closer.Close() })

	// Accept in background and capture first line.
	accepted := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			accepted <- ""
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 512)
		n, _ := conn.Read(buf)
		accepted <- string(buf[:n])
		_ = conn.Close()
	}()

	logger.Infof("cli", "hello")

	select {
	case got := <-accepted:
		if got == "" {
			t.Fatal("sink received no data")
		}
		// Must be a valid JSON event with correct service.
		var evt testEvent
		if err := json.Unmarshal([]byte(got[:len(got)-1]), &evt); err != nil { // strip trailing newline
			t.Fatalf("decode sink payload: %v (raw %q)", err, got)
		}
		if evt.Service != "orchv3" {
			t.Fatalf("sink event.service = %q, want orchv3", evt.Service)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for sink delivery")
	}

	// Stderr must also have received the event.
	if !bytes.Contains(stderr.Bytes(), []byte(`"service":"orchv3"`)) {
		t.Fatalf("stderr missing event: %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/orchv3/ -run TestBuildLogger -v`
Expected: FAIL — `buildLogger` не определён.

- [ ] **Step 3: Create logger_setup.go**

Создать `cmd/orchv3/logger_setup.go`:

```go
package main

import (
	"io"

	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

func buildLogger(stderr io.Writer, cfg config.Config, warnOut io.Writer) (steplog.Logger, io.Closer, error) {
	if cfg.Logstash.Addr == "" {
		return steplog.NewWithService(stderr, cfg.AppName), nil, nil
	}

	sink := steplog.NewTCPSink(
		cfg.Logstash.Addr,
		cfg.Logstash.BufferSize,
		cfg.Logstash.DialTimeout,
		warnOut,
	)

	out := io.MultiWriter(stderr, sink)
	return steplog.NewWithService(out, cfg.AppName), sink, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/orchv3/ -run TestBuildLogger -v -timeout 15s`
Expected: PASS.

- [ ] **Step 5: Run whole cmd package**

Run: `go test ./cmd/orchv3/ -v -timeout 30s`
Expected: все тесты — PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/orchv3/logger_setup.go cmd/orchv3/logger_setup_test.go
git commit -m "feat(orchv3): buildLogger helper with optional TCP sink"
```

---

## Task 9: Подключить `buildLogger` в `main.go`

Заменить создание логгера в `run()` на вызов `buildLogger`. `defer` закрытия sink-а. Внутренний warn-канал — прямой `os.Stderr`, чтобы не зациклиться на себя.

**Files:**
- Modify: `cmd/orchv3/main.go`

- [ ] **Step 1: Run existing tests to baseline**

Run: `go test ./cmd/orchv3/ -v -timeout 30s`
Expected: все существующие тесты (включая TestBuildLogger из Task 8) — PASS.

- [ ] **Step 2: Modify run() in main.go**

Заменить строки `cmd/orchv3/main.go:20-28` (от начала `run` до создания `logger`) на:

```go
func run(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		steplog.New(stderr).Errorf("cli", "load config: %v", err)
		return 1
	}

	logger, closer, err := buildLogger(stderr, cfg, os.Stderr)
	if err != nil {
		steplog.New(stderr).Errorf("cli", "build logger: %v", err)
		return 1
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}
```

Остальная часть `run()` не меняется — она уже использует переменную `logger`.

- [ ] **Step 3: Run all tests**

Run: `go test ./... -timeout 60s`
Expected: все тесты — PASS.

- [ ] **Step 4: Run go fmt**

Run: `go fmt ./...`
Expected: пустой вывод (файлы уже отформатированы).

- [ ] **Step 5: Smoke build**

Run: `go build ./...`
Expected: PASS без ошибок.

- [ ] **Step 6: Commit**

```bash
git add cmd/orchv3/main.go
git commit -m "feat(orchv3): wire TCPSink into main via buildLogger"
```

---

## Task 10: Обновить `.env.example`

**Files:**
- Modify: `.env.example`

- [ ] **Step 1: Append ELK section**

Файл `.env.example`, дописать в конец:

```
# ELK integration
LOGSTASH_ADDR=
LOGSTASH_BUFFER_SIZE=
LOGSTASH_DIAL_TIMEOUT=
```

- [ ] **Step 2: Verify file content**

Run: `cat .env.example`
Expected: файл содержит старые секции + новую «ELK integration» с тремя пустыми ключами.

- [ ] **Step 3: Commit**

```bash
git add .env.example
git commit -m "docs(env): document LOGSTASH_* variables in .env.example"
```

---

## Task 11: `deploy/docker-compose.yml` + Logstash pipeline

Поднимаем ES/Logstash/Kibana в single-node dev-режиме. Порт Logstash наружу только на `127.0.0.1:5000`.

**Files:**
- Create: `deploy/docker-compose.yml`
- Create: `deploy/logstash/pipeline/orchv3.conf`

- [ ] **Step 1: Create pipeline config**

Создать `deploy/logstash/pipeline/orchv3.conf`:

```
input {
  tcp {
    port  => 5000
    codec => json_lines
  }
}

filter {
  date {
    match        => ["time", "ISO8601"]
    target       => "@timestamp"
    remove_field => ["time"]
  }
  mutate {
    rename => { "type" => "level" }
  }
}

output {
  elasticsearch {
    hosts => ["http://elasticsearch:9200"]
    index => "orchv3-%{+YYYY.MM.dd}"
  }
}
```

- [ ] **Step 2: Create docker-compose.yml**

Создать `deploy/docker-compose.yml`:

```yaml
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.13.4
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    volumes:
      - es-data:/usr/share/elasticsearch/data
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:9200/_cluster/health?wait_for_status=yellow&timeout=5s"]
      interval: 5s
      timeout: 10s
      retries: 30
    ports:
      - "127.0.0.1:9200:9200"

  logstash:
    image: docker.elastic.co/logstash/logstash:8.13.4
    environment:
      - LS_JAVA_OPTS=-Xms256m -Xmx256m
    volumes:
      - ./logstash/pipeline:/usr/share/logstash/pipeline:ro
    depends_on:
      elasticsearch:
        condition: service_healthy
    ports:
      - "127.0.0.1:5000:5000"

  kibana:
    image: docker.elastic.co/kibana/kibana:8.13.4
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    depends_on:
      elasticsearch:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:5601/api/status | grep -q '\"level\":\"available\"'"]
      interval: 5s
      timeout: 10s
      retries: 60
    ports:
      - "127.0.0.1:5601:5601"

  kibana-setup:
    image: curlimages/curl:8.7.1
    depends_on:
      kibana:
        condition: service_healthy
    volumes:
      - ./kibana/setup.sh:/setup.sh:ro
      - ./kibana/saved-objects.ndjson:/saved-objects.ndjson:ro
    entrypoint: ["sh", "/setup.sh"]
    restart: "no"

volumes:
  es-data:
```

- [ ] **Step 3: Validate compose file syntax**

Run: `docker compose -f deploy/docker-compose.yml config --quiet`
Expected: пустой вывод (валидный YAML, схема compose ок). Если Docker не установлен в CI/локально — шаг пропускается, фиксируется в отчёте.

- [ ] **Step 4: Verify no `:latest` image tags**

Run: `grep -n 'image:' deploy/docker-compose.yml | grep -i ':latest' || echo OK_NO_LATEST`
Expected: `OK_NO_LATEST`. Все tags — конкретные версии (acceptance criterion #12).

- [ ] **Step 5: Commit**

```bash
git add deploy/docker-compose.yml deploy/logstash/pipeline/orchv3.conf
git commit -m "feat(deploy): docker-compose for ES/Logstash/Kibana + orchv3 pipeline"
```

---

## Task 12: Kibana provisioning — setup.sh + saved-objects.ndjson (data view)

Скелет `saved-objects.ndjson` содержит data view `orchv3-*`. Дашборд добавляется follow-up-ом после ручного создания в Kibana UI и экспорта.

**Files:**
- Create: `deploy/kibana/setup.sh`
- Create: `deploy/kibana/saved-objects.ndjson`

- [ ] **Step 1: Create setup.sh**

Создать `deploy/kibana/setup.sh`:

```sh
#!/bin/sh
set -e

until curl -sf http://kibana:5601/api/status > /dev/null; do
  echo "waiting for kibana..."
  sleep 2
done

echo "importing saved objects..."
curl -sf -X POST "http://kibana:5601/api/saved_objects/_import?overwrite=true" \
  -H "kbn-xsrf: true" \
  --form file=@/saved-objects.ndjson

echo "kibana setup complete"
```

- [ ] **Step 2: Create saved-objects.ndjson with data view only**

Создать `deploy/kibana/saved-objects.ndjson` (одна строка JSON на объект, заканчиваем переводом строки):

```
{"attributes":{"title":"orchv3-*","timeFieldName":"@timestamp","name":"orchv3"},"id":"orchv3-data-view","managed":false,"references":[],"type":"index-pattern","version":"1"}
{"excludedObjects":[],"excludedObjectsCount":0,"exportedCount":1,"missingRefCount":0,"missingReferences":[]}
```

- [ ] **Step 3: Validate ndjson is line-delimited**

Run: `wc -l deploy/kibana/saved-objects.ndjson`
Expected: `2 deploy/kibana/saved-objects.ndjson` (2 строки).

- [ ] **Step 4: Make setup.sh executable (for POSIX checkouts)**

Run: `git update-index --chmod=+x deploy/kibana/setup.sh`
(На Windows это пропишется при следующем checkout'е; скрипт в контейнере всё равно запускается через `sh /setup.sh`, поэтому mode не критичен для работы.)

- [ ] **Step 5: Commit**

```bash
git add deploy/kibana/setup.sh deploy/kibana/saved-objects.ndjson
git commit -m "feat(deploy): Kibana auto-provisioning with orchv3-* data view"
```

---

## Task 13: `docs/elk-demo.md` — how-to и smoke-test

**Files:**
- Create: `docs/elk-demo.md`

- [ ] **Step 1: Create elk-demo.md**

Создать `docs/elk-demo.md` (внешний fence для плана — `~~~`, внутренние — обычные ` ``` ` для консольных блоков):

~~~markdown
# ELK demo — как поднять и что показать

## Требования

- Docker / Docker Desktop.
- ≥ 2 GB свободной RAM на docker-хосте (ES под JVM + Logstash + Kibana).
- Порты `5000`, `5601`, `9200` свободны на `127.0.0.1`.

## Запуск стека

```
docker compose -f deploy/docker-compose.yml up -d
```

Ждём, пока `kibana-setup` выйдет с кодом 0:

```
docker compose -f deploy/docker-compose.yml logs -f kibana-setup
```

После этого в Kibana (`http://localhost:5601`) создан data view `orchv3-*`.

## Проверка health

- Elasticsearch: `curl -sf http://127.0.0.1:9200/_cluster/health?pretty`
- Kibana: `curl -sf http://127.0.0.1:5601/api/status | jq '.status.overall'`

## Остановка

Мягкая остановка (данные ES сохраняются в volume):

```
docker compose -f deploy/docker-compose.yml down
```

## Полный сброс данных (чистый лист перед демо)

```
docker compose -f deploy/docker-compose.yml down -v
```

Флаг `-v` удаляет именованный volume `es-data`. Следующий `up -d` начнёт с пустого индекса.

## Smoke-test (без оркестратора)

```
echo '{"time":"2026-04-24T10:00:00Z","service":"orchv3","module":"smoke","type":"info","message":"hello"}' \
  | nc 127.0.0.1 5000
```

Через пару секунд событие видно в Kibana Discover, индекс `orchv3-*`.

## Запуск оркестратора с доставкой в ELK

В `.env`:

```
LOGSTASH_ADDR=127.0.0.1:5000
```

Запуск бинаря — как обычно. Поле `service` берётся из `APP_NAME`.

Если `LOGSTASH_ADDR` пусто — sink выключен, бинарь работает как раньше (только stderr).

## Экспорт дашборда после ручной настройки

1. В Kibana создать дашборд «orchv3 live» (timeline, modules × levels, recent errors).
2. Management → Stack Management → Saved Objects → Export (Data views + Dashboards).
3. Полученный `.ndjson` сохранить как `deploy/kibana/saved-objects.ndjson` (заменить).
4. `docker compose -f deploy/docker-compose.yml up -d --force-recreate kibana-setup` — идемпотентный reimport.

## Smoke-проверка чистоты stderr

На типичном прогоне каждая строка stderr должна быть валидным JSON:

```
orchv3 <args> 2> /tmp/run.log
jq -c . < /tmp/run.log
```

`jq` не должен выдавать ошибок.
~~~

- [ ] **Step 2: Commit**

```bash
git add docs/elk-demo.md
git commit -m "docs: add ELK demo how-to and smoke-test"
```

---

## Task 14: Итоговый прогон + acceptance-чеклист

**Files:**
- (проверки, без изменений в коде)

- [ ] **Step 1: go fmt**

Run: `go fmt ./...`
Expected: пустой вывод.

- [ ] **Step 2: go test полный**

Run: `go test ./... -timeout 120s`
Expected: все пакеты — ok, включая `internal/steplog` (delivery + non-blocking + drops + reconnect + close-flush + drop-warning) и `cmd/orchv3` (BuildLogger disabled/enabled).

- [ ] **Step 3: go build**

Run: `go build ./...`
Expected: бинарь собирается без ошибок.

- [ ] **Step 4: Acceptance-чеклист (ручной, если Docker доступен)**

Проверить по пунктам спеки `docs/specs/2026-04-24-elk-logging-design.md` раздел «Критерии приёмки»:

- [ ] №1: `go fmt ./...` и `go test ./...` зелёные — сделано на шагах 1–2.
- [ ] №2: тесты TCPSink покрывают delivery/reconnect/overflow/non-blocking/close-flush — сделано в задачах 3–7.
- [ ] №3: `docker compose -f deploy/docker-compose.yml up -d` поднимает ES/Logstash/Kibana, `kibana-setup` exit 0.
- [ ] №4: data view `orchv3-*` виден в Kibana после setup. Дашборд — follow-up.
- [ ] №5: запуск бинаря с `LOGSTASH_ADDR=127.0.0.1:5000` даёт события в Kibana <5s.
- [ ] №6: запуск с пустым `LOGSTASH_ADDR` — stderr-JSON как раньше, без warning.
- [ ] №7: бинарь поднят до ELK — не падает; после старта ELK доставка начинается.
- [ ] №8: остановка Logstash во время работы — бинарь не падает и не виснет; при рестарте Logstash доставка возобновляется.
- [ ] №9: `.env.example` содержит три новых ключа пустыми.
- [ ] №10: `docs/elk-demo.md` содержит старт/остановку, health-checks, nc-smoke, URL дашборда, процедуру `docker compose down -v` и минимальные требования к памяти.
- [ ] №11: `jq -c .` на stderr-логе типичного прогона не даёт ошибок.
- [ ] №12: `grep -n 'image:' deploy/docker-compose.yml | grep -i ':latest'` возвращает пустой результат — все tags зафиксированы конкретной версией.

- [ ] **Step 5: Summary commit (опционально)**

Если по ходу ручной проверки были мелкие правки — собрать их в один финальный коммит:

```bash
git add -u
git commit -m "chore(elk): acceptance fixes after manual verification"
```

Иначе шаг пропускается.
