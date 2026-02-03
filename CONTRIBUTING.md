# Contributing to gogpu/gg-svg

Thank you for your interest in contributing to **gg-svg** — the SVG export backend for gg's recording system!

## Requirements

- **Go 1.25+** (required for modern features)
- **golangci-lint** for code quality checks
- **git** with conventional commits knowledge

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/gg-svg`
3. Create a branch: `git checkout -b feat/your-feature`
4. Make your changes
5. Run tests: `go test ./...`
6. Run linter: `golangci-lint run --timeout=5m`
7. Commit: `git commit -m "feat: add your feature"`
8. Push: `git push origin feat/your-feature`
9. Open a Pull Request

## Development Setup

```bash
# Clone the repository
git clone https://github.com/gogpu/gg-svg
cd gg-svg

# Install dependencies
go mod download

# Run tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run linter
golangci-lint run --timeout=5m

# Format code
go fmt ./...
```

## Architecture

gg-svg implements the `recording.Backend` interface from gogpu/gg to render vector graphics to SVG format.

Key features:
- Pure Go implementation (no external dependencies beyond gg)
- Standard SVG 1.1 output
- Auto-registration via `init()` for blank import pattern

## Code Style

- **Formatting:** `gofmt` (run `go fmt ./...` before committing)
- **Linting:** `golangci-lint` with project configuration
- **Coverage:** Minimum 70% for new code
- **Documentation:** All public APIs must be documented

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `test` | Adding or fixing tests |
| `refactor` | Code change without feature/fix |
| `perf` | Performance improvement |
| `chore` | Maintenance, dependencies |

### Examples

```bash
feat: add gradient spread mode support
fix: resolve XML escaping in text content
docs: update README with gradient example
test: add edge case tests for path operations
```

## Pull Request Guidelines

### Before Opening a PR

1. **Ensure all tests pass:** `go test -race ./...`
2. **Check linter:** `golangci-lint run --timeout=5m`
3. **Format code:** `go fmt ./...`
4. **Update documentation** if adding/changing public APIs

### PR Requirements

- **Focused:** One feature or fix per PR
- **Tested:** Include tests for new functionality
- **Documented:** Update relevant docs
- **CI passing:** All GitHub Actions checks must pass

## Reporting Issues

When opening an issue, please include:

- **Go version:** `go version`
- **OS and architecture:** e.g., Windows 11 x64, macOS 14 ARM64
- **gg-svg version:** e.g., v0.1.0
- **Minimal reproduction:** Code snippet
- **Expected vs actual behavior**
- **Error messages and stack traces**
- **Output SVG** (if visual issue)

## Questions?

- **GitHub Discussions:** For questions and ideas
- **GitHub Issues:** For bugs and feature requests

---

Thank you for contributing to gogpu/gg-svg!
