# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-03

### Added

- **SVG Backend** for gg's recording system
  - `Backend` — implements `recording.Backend`, `recording.WriterBackend`, `recording.FileBackend`
  - Auto-registration via blank import (`import _ "github.com/gogpu/gg-svg"`)
  - Solid color fills and strokes
  - Linear and radial gradients (with spread modes)
  - Path operations (fill, stroke, clip)
  - Transformations (matrix)
  - Stroke styles (width, cap, join, dash patterns)
  - State management (Save/Restore via groups)
  - Text rendering
  - Image embedding (PNG data URI)
  - XML escaping for security

- **Output Format**
  - Standard SVG 1.1 with XML declaration
  - SVG namespace
  - Definitions section for gradients and clip paths
  - Proper attribute encoding

- **Project Infrastructure**
  - LICENSE (MIT)
  - CONTRIBUTING.md
  - CODE_OF_CONDUCT.md
  - SECURITY.md
  - GitHub Actions CI (build, test, lint on Linux/macOS/Windows)
  - golangci-lint configuration

### Notes

- Pure Go implementation (no external dependencies beyond gg)
- Sweep gradients fallback to first stop color (SVG limitation)
- Text uses default font (custom font embedding not supported)
- Images are embedded as PNG data URIs (increases file size)
- Part of the [gogpu](https://github.com/gogpu) ecosystem

[0.1.0]: https://github.com/gogpu/gg-svg/releases/tag/v0.1.0
