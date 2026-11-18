# Contributing to `redis-top`

Thank you for taking the time to contribute to `redis-top`! We welcome bug reports, feature requests, and pull requests from everyone.

## Development Workflow

1. **Fork and Clone**:
   ```bash
   git clone https://github.com/tarun1357/redis-top.git
   cd redis-top
   ```

2. **Run Tests**:
   `redis-top` uses zero external dependencies in its core engine so it builds instantaneously.
   ```bash
   make test
   ```

3. **Test Demo Mode Locally**:
   You can verify terminal rendering and prefix aggregation without a local Redis server using `--demo`:
   ```bash
   make run-demo
   ```

4. **Code Style & Standards**:
   - Ensure `go fmt ./...` and `go vet ./...` pass with zero warnings.
   - Add table-driven unit tests for any new prefix parser, RESP command, or memory formatter.
   - Keep external dependencies to an absolute minimum.

## Submitting a Pull Request

- Reference any relevant issue numbers in your PR description.
- Ensure the GitHub Actions CI workflow passes on your branch.
