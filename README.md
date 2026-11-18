# `redis-top` — Zero-Dependency, Production-Safe Redis Memory & Prefix Analyzer

[![CI](https://github.com/tarun1357/redis-top/actions/workflows/ci.yml/badge.svg)](https://github.com/tarun1357/redis-top/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tarun1357/redis-top)](https://goreportcard.com/report/github.com/tarun1357/redis-top)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`redis-top` is a fast, zero-dependency CLI utility written in Go that scans Redis instances using non-blocking cursor iteration (`SCAN`) and pipelined `MEMORY USAGE` + `TTL` inspection. It groups keys by namespace prefix (e.g., `session:user:*`, `cache:feed:*`) to immediately surface memory hotspots, missing TTLs (memory leaks), and key-count skew without blocking production traffic.

---

## Why `redis-top`?

When Redis memory spikes in production, running `KEYS *` or `redis-cli --bigkeys` either blocks the single-threaded event loop or only shows individual largest keys—missing the fact that **400,000 tiny keys under `otp:verify:*` without a TTL** are consuming 65% of your RAM.

- **Production-Safe by Design:** Uses cursor-based `SCAN` with configurable batch sizes and sleep throttling (`--throttle-ms`) to guarantee zero latency spikes on live workloads.
- **Pipelined RESP Wire Client:** Built on a custom, zero-dependency RESP (Redis Serialization Protocol) pipelining client that inspects thousands of keys per second over a single TCP socket.
- **Missing-TTL Leak Detection:** Highlights prefixes where keys lack an expiration policy (`TTL = -1`), making it trivial to spot unbounded cache growth.
- **Multiple Output Formats:** Renders clean ANSI terminal tables or structured `JSON` for CI/CD pipelines, Datadog/Grafana ingestion, or automated alerting.

---

## Quick Preview

```text
$ redis-top --addr 127.0.0.1:6379 --sample 10000 --depth 2

redis-top v0.1.0 — Connected to 127.0.0.1:6379 (DB 0)
Scanned 10,000 keys in 142ms (Batch: 250, Depth: 2, Delimiters: ":")

PREFIX                 KEYS    % KEYS   EST. MEMORY   % MEM    AVG SIZE   NO-TTL (%)   AVG TTL
------------------------------------------------------------------------------------------------
session:auth:*         4,120   41.2%    31.48 MB      48.9%    7.82 KB    0.0%         48m12s
cache:vehicle_pass:*   2,890   28.9%    18.92 MB      29.4%    6.70 KB    94.2% [!]    12m00s
ratelimit:api:*        1,940   19.4%    3.12 MB       4.8%     1.65 KB    0.0%         45s
queue:dlq:*            640     6.4%     9.84 MB       15.3%    15.74 KB   100.0% [!]   -
user:profile:*         410     4.1%     1.04 MB       1.6%     2.60 KB    12.5%        6h15m0s
------------------------------------------------------------------------------------------------
TOTAL                  10,000  100.0%   64.40 MB      100.0%   6.60 KB    35.3%        -

[!] Warning: 2 prefixes have >80% keys without a TTL (potential unbounded memory growth).
```

---

## Installation

### Via `go install` (Recommended)
```bash
go install github.com/tarun1357/redis-top/cmd/redis-top@latest
```

### Build from Source
```bash
git clone https://github.com/tarun1357/redis-top.git
cd redis-top
make build
./bin/redis-top --demo
```

---

## Usage & Flags

```bash
redis-top [flags]
```

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--addr` | `127.0.0.1:6379` | Target Redis server `host:port` |
| `--password` | `""` | Redis password (or set `REDIS_PASSWORD` env var) |
| `--db` | `0` | Redis database index to inspect |
| `--match` | `*` | Glob-style pattern filter passed to `SCAN` |
| `--sample` | `5000` | Maximum number of keys to sample (`0` for full scan) |
| `--count` | `200` | `SCAN COUNT` hint per iteration batch |
| `--depth` | `2` | Prefix hierarchy depth (e.g., `a:b:c` at depth 2 -> `a:b:*`) |
| `--sep` | `:` | Delimiter characters used to split key namespaces |
| `--top` | `15` | Number of top prefixes to display sorted by memory |
| `--throttle-ms` | `0` | Sleep duration in milliseconds between pipelined batches |
| `--json` | `false` | Emit machine-readable JSON summary instead of ASCII table |
| `--demo` | `false` | Run with synthetic workload data (no Redis server needed) |

### Examples

**1. Inspect top 20 namespaces up to depth 3 on production with 5ms batch throttling:**
```bash
redis-top --addr prod-redis.internal:6379 --sample 25000 --depth 3 --top 20 --throttle-ms 5
```

**2. Drill down into a specific prefix (`cache:*`) and export as JSON:**
```bash
redis-top --match "cache:*" --depth 3 --json > cache_report.json
```

**3. Try the interactive demo mode without a live Redis instance:**
```bash
redis-top --demo
```

---

## Architecture

```text
+-------------------+       SCAN 0 COUNT 200        +--------------------+
|                   | ----------------------------> |                    |
|   redis-top CLI   |                               |    Redis Server    |
|  (Scanner Engine) | <---------------------------- | (Non-Blocking I/O) |
|                   |   Pipeline: MEMORY USAGE/TTL  |                    |
+---------+---------+                               +--------------------+
          |
          v
+-------------------+       Prefix Tokenizer        +--------------------+
|   Key Inspector   | ----------------------------> |  Namespace Aggreg. |
| (Sampled Metrics) |   (Depth & Delimiter Split)   | (Heap Sort & Leak) |
+-------------------+                               +--------------------+
```

---

## Roadmap

- [x] Zero-dependency pipelined RESP wire protocol client
- [x] Configurable namespace depth and multi-character delimiter extraction
- [x] No-TTL memory leak warning heuristics
- [x] JSON output mode for CI/monitoring integration
- [ ] Redis Cluster auto-discovery (`CLUSTER SLOTS` fan-out)
- [ ] Interactive TUI mode with live auto-refresh

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a Pull Request.

## License

Licensed under the [MIT License](LICENSE).
