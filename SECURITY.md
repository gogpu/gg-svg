# Security Policy

## Supported Versions

gogpu/gg-svg is currently in early development (v0.x.x).

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1.0 | :x:                |

## Reporting a Vulnerability

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security issues via:

1. **Private Security Advisory** (preferred):
   https://github.com/gogpu/gg-svg/security/advisories/new

2. **GitHub Discussions** (for less critical issues):
   https://github.com/gogpu/gg-svg/discussions

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Potential impact

### Response Timeline

- **Initial Response**: Within 72 hours
- **Fix & Disclosure**: Coordinated with reporter

## Security Considerations

gogpu/gg-svg is an SVG generation library. Security considerations:

1. **File System** — SaveToFile writes to specified paths
2. **XML Injection** — Text content is properly escaped
3. **Memory** — Large canvases allocate significant memory

## Security Contact

- **GitHub Security Advisory**: https://github.com/gogpu/gg-svg/security/advisories/new
- **Public Issues**: https://github.com/gogpu/gg-svg/issues

---

**Thank you for helping keep gogpu/gg-svg secure!**
