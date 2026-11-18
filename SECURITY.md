# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

Because `redis-top` connects to production and staging Redis instances, we take credential handling and read-only safety seriously:
- `redis-top` strictly executes non-mutating inspection commands (`SCAN`, `MEMORY USAGE`, `TTL`, `TYPE`).
- Please do **not** report security vulnerabilities through public GitHub issues. Open a draft security advisory on GitHub instead.
