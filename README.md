# health_status

[English](#english) · [Español](#espanol)

<a id="english"></a>

A small service-health monitor written in Go. It polls a set of HTTP
endpoints on independent schedules and raises an alert when a service changes
state, with a re-check step that filters out flapping so a single blip never
pages anyone.

## Features

- **One goroutine per service**, each on its own `time.Ticker` interval.
- **State-change alerts only.** It stays quiet while a service is healthy and
  fires once when it goes down (and again when it recovers).
- **Flap filtering.** Before alerting, a change is re-checked a configurable
  number of times; a transient blip that doesn't hold is ignored.
- **Dependency checks.** A service can declare a `dependencies_url` that
  returns a `{ name: state }` map (e.g. databases). It's polled while the
  service is up and alerts independently when a dependency degrades.
- **Pluggable alert channels** behind a one-method `Notifier` interface. Ships
  with a console channel, a generic webhook (Slack/Discord compatible) and a
  Telegram channel; adding a new channel never touches the monitor.

## Quickstart

The repo ships with a mock service so you can watch the whole flow without
wiring up anything real.

```bash
docker compose up --build
```

This starts:

- `micro-status` - a mock HTTP service on `:8081`.
- `monitor` - watches it and prints health on the console.

In another terminal, flip the service to failing and watch the outage and
recovery alerts roll in:

```bash
curl localhost:8081/toggle   # service goes DOWN  -> outage alert
curl localhost:8081/toggle   # service goes UP    -> recovery alert
```

To see a dependency alert (the service stays healthy, but one of its
dependencies degrades):

```bash
curl localhost:8081/toggle/redis   # redis -> error -> dependency alert
curl localhost:8081/toggle/redis   # redis -> ok    -> dependency recovery
```

## Running locally

```bash
# Terminal 1: the mock service
go run ./cmd/mockservice

# Terminal 2: the monitor (uses config.yaml -> localhost:8081)
go run ./cmd
```

## Configuration

Services and the optional webhook live in `config.yaml`:

```yaml
services:
  - name: Orders API
    url: http://localhost:8081/health
    interval: 5s
    dependencies_url: http://localhost:8081/health/dependencies  # optional
  - name: Public Echo
    url: https://httpbin.org/status/200
    interval: 10s

webhook:
  url: ""   # Slack/Discord incoming webhook; empty = console only
```

Timing knobs come from the environment (see `.env.example`); all have sane
defaults, so a missing `.env` is fine:

| Variable                | Default | Meaning                                  |
| ----------------------- | ------- | ---------------------------------------- |
| `REQUEST_TIMEOUT`       | `5s`    | Per-request HTTP timeout                 |
| `DOWN_CONFIRM_ATTEMPTS` | `2`     | Re-checks before confirming an outage    |
| `DOWN_CONFIRM_DELAY`    | `1s`    | Wait between outage re-checks            |
| `UP_CONFIRM_ATTEMPTS`   | `1`     | Re-checks before confirming a recovery   |
| `UP_CONFIRM_DELAY`      | `10s`   | Wait between recovery re-checks          |

To also send alerts to Telegram, set both `TELEGRAM_BOT_TOKEN` and
`TELEGRAM_CHAT_ID`. When either is empty the channel stays off.

## Layout

```
cmd/
  main.go             monitor entrypoint
  mockservice/        toggleable HTTP service for local testing
internal/
  config/             YAML + env config loading
  service/            HTTP health + dependency checks
  monitor/            polling loop, flap filtering, alert routing
  notify/             Notifier interface impls (console, webhook, telegram)
```

## License

MIT — see [LICENSE](LICENSE).

---

<a id="espanol"></a>

## Español

Un pequeño monitor de salud de servicios escrito en Go. Chequea un conjunto de
endpoints HTTP en horarios independientes y dispara una alerta cuando un
servicio cambia de estado, con un paso de re-chequeo que filtra el flapping para
que un solo parpadeo nunca despierte a nadie.

### Características

- **Una goroutine por servicio**, cada una con su propio intervalo (`time.Ticker`).
- **Alertas solo en cambio de estado.** Se queda callado mientras el servicio
  está sano y avisa una vez cuando cae (y otra cuando se recupera).
- **Filtro de flapping.** Antes de alertar, el cambio se re-chequea una cantidad
  configurable de veces; un parpadeo pasajero que no se sostiene se ignora.
- **Chequeo de dependencias.** Un servicio puede declarar un `dependencies_url`
  que devuelve un mapa `{ name: state }` (ej. bases de datos). Se consulta
  mientras el servicio está arriba y alerta por separado cuando una dependencia
  se degrada.
- **Canales de alerta pluggables** detrás de una interfaz `Notifier` de un solo
  método. Trae consola, un webhook genérico (compatible con Slack/Discord) y un
  canal de Telegram; agregar un canal nuevo nunca toca el monitor.

### Inicio rápido

El repo incluye un servicio mock para ver todo el flujo sin cablear nada real.

```bash
docker compose up --build
```

Esto levanta:

- `micro-status` - un servicio HTTP mock en `:8081`.
- `monitor` - lo vigila e imprime la salud por consola.

En otra terminal, forzá la falla del servicio y mirá llegar las alertas de caída
y recuperación:

```bash
curl localhost:8081/toggle   # el servicio CAE   -> alerta de caída
curl localhost:8081/toggle   # el servicio SUBE  -> alerta de recuperación
```

Para ver una alerta de dependencia (el servicio sigue sano, pero una de sus
dependencias se degrada):

```bash
curl localhost:8081/toggle/redis   # redis -> error -> alerta de dependencia
curl localhost:8081/toggle/redis   # redis -> ok    -> recuperación de dependencia
```

### Ejecutar localmente

```bash
# Terminal 1: el servicio mock
go run ./cmd/mockservice

# Terminal 2: el monitor (usa config.yaml -> localhost:8081)
go run ./cmd
```

### Configuración

Los servicios y el webhook opcional viven en `config.yaml`:

```yaml
services:
  - name: Orders API
    url: http://localhost:8081/health
    interval: 5s
    dependencies_url: http://localhost:8081/health/dependencies  # opcional
  - name: Public Echo
    url: https://httpbin.org/status/200
    interval: 10s

webhook:
  url: ""   # incoming webhook de Slack/Discord; vacío = solo consola
```

Los parámetros de timing vienen del entorno (ver `.env.example`); todos tienen
defaults razonables, así que un `.env` ausente no es problema:

| Variable                | Default | Significado                              |
| ----------------------- | ------- | ---------------------------------------- |
| `REQUEST_TIMEOUT`       | `5s`    | Timeout HTTP por request                 |
| `DOWN_CONFIRM_ATTEMPTS` | `2`     | Re-chequeos antes de confirmar una caída |
| `DOWN_CONFIRM_DELAY`    | `1s`    | Espera entre re-chequeos de caída        |
| `UP_CONFIRM_ATTEMPTS`   | `1`     | Re-chequeos antes de confirmar recuperación |
| `UP_CONFIRM_DELAY`      | `10s`   | Espera entre re-chequeos de recuperación |

Para enviar alertas también a Telegram, definí `TELEGRAM_BOT_TOKEN` y
`TELEGRAM_CHAT_ID`. Si cualquiera de los dos está vacío, el canal queda apagado.

### Estructura

```
cmd/
  main.go             entrypoint del monitor
  mockservice/        servicio HTTP conmutable para pruebas locales
internal/
  config/             carga de config YAML + env
  service/            chequeos HTTP de salud + dependencias
  monitor/            loop de polling, filtro de flapping, ruteo de alertas
  notify/             implementaciones de la interfaz Notifier (consola, webhook, telegram)
```

## Licencia

MIT — ver [LICENSE](LICENSE).
