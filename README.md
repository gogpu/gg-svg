# gg-svg

SVG export backend for [gg](https://github.com/gogpu/gg)'s recording system.

Part of the [GoGPU](https://github.com/gogpu) ecosystem.

## Installation

```bash
go get github.com/gogpu/gg-svg
```

## Usage

Import with a blank identifier to register the SVG backend:

```go
import (
    "github.com/gogpu/gg/recording"
    _ "github.com/gogpu/gg-svg" // Register SVG backend
)

func main() {
    // Create a recorder
    rec := recording.NewRecorder(800, 600)

    // Draw something
    rec.SetFillRGBA(1, 0, 0, 1) // Red
    rec.DrawRectangle(100, 100, 200, 150)
    rec.Fill()

    // Finish recording
    r := rec.FinishRecording()

    // Create SVG backend
    backend, err := recording.NewBackend("svg")
    if err != nil {
        log.Fatal(err)
    }

    // Playback to SVG
    r.Playback(backend)

    // Save to file
    if fb, ok := backend.(recording.FileBackend); ok {
        fb.SaveToFile("output.svg")
    }
}
```

## Features

- Solid color fills and strokes
- Linear and radial gradients (with spread modes)
- Path operations (fill, stroke, clip)
- Transformations (matrix)
- Stroke styles (width, cap, join, dash patterns)
- State management (Save/Restore via groups)
- Text rendering
- Image embedding (PNG data URI)
- XML escaping for security

## Output Format

The backend generates standard SVG 1.1 with:
- XML declaration
- SVG namespace
- Definitions section for gradients and clip paths
- Proper attribute encoding

Example output:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="800" height="600" viewBox="0 0 800 600">
<defs>...</defs>
<path d="M100 100L300 100L300 250L100 250Z" fill="rgb(255,0,0)" stroke="none"/>
</svg>
```

## Limitations

- Sweep gradients fallback to first stop color (SVG limitation)
- Text uses default font (custom font embedding not supported)
- Images are embedded as PNG data URIs (increases file size)

## License

MIT License - see [LICENSE](LICENSE) for details.
