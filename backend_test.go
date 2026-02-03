package svg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/recording"
)

func TestBackendRegistration(t *testing.T) {
	// Test that the SVG backend is registered
	if !recording.IsRegistered("svg") {
		t.Error("SVG backend should be registered")
	}

	// Test that we can create a backend
	backend, err := recording.NewBackend("svg")
	if err != nil {
		t.Fatalf("Failed to create SVG backend: %v", err)
	}

	if backend == nil {
		t.Error("Backend should not be nil")
	}
}

func TestBackendInterfaces(t *testing.T) {
	backend := NewBackend()

	// Test Backend interface
	var _ recording.Backend = backend

	// Test WriterBackend interface
	var _ recording.WriterBackend = backend

	// Test FileBackend interface
	var _ recording.FileBackend = backend
}

func TestBackendLifecycle(t *testing.T) {
	backend := NewBackend()

	// Test Begin
	err := backend.Begin(800, 600)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Test End
	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}
}

func TestBackendSaveRestore(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(800, 600)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Save state
	backend.Save()

	// Set transform
	backend.SetTransform(recording.Translate(100, 100))

	// Restore should work without error
	backend.Restore()

	// Multiple saves and restores
	backend.Save()
	backend.Save()
	backend.Restore()
	backend.Restore()

	// Restore with empty stack should be no-op
	backend.Restore() // Should not panic

	_ = backend.End()
}

func TestBackendFillPath(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create a simple rectangle path
	path := gg.NewPath()
	path.Rectangle(50, 50, 100, 80)

	// Create a solid brush
	brush := recording.NewSolidBrush(gg.RGBA{R: 1, G: 0, B: 0, A: 1})

	// Fill the path
	backend.FillPath(path, brush, recording.FillRuleNonZero)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Write to buffer to verify no errors
	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<svg") {
		t.Error("Output should contain SVG element")
	}
	if !strings.Contains(svg, "<path") {
		t.Error("Output should contain path element")
	}
	if !strings.Contains(svg, `fill="rgb(255,0,0)"`) {
		t.Error("Output should contain red fill color")
	}
}

func TestBackendStrokePath(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create a triangle path
	path := gg.NewPath()
	path.MoveTo(100, 50)
	path.LineTo(150, 150)
	path.LineTo(50, 150)
	path.Close()

	// Create brush and stroke
	brush := recording.NewSolidBrush(gg.RGBA{R: 0, G: 0, B: 1, A: 1})
	stroke := recording.Stroke{
		Width:      2.0,
		Cap:        recording.LineCapRound,
		Join:       recording.LineJoinRound,
		MiterLimit: 4.0,
	}

	backend.StrokePath(path, brush, stroke)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `stroke="rgb(0,0,255)"`) {
		t.Error("Output should contain blue stroke color")
	}
	if !strings.Contains(svg, `stroke-width="2"`) {
		t.Error("Output should contain stroke width")
	}
	if !strings.Contains(svg, `stroke-linecap="round"`) {
		t.Error("Output should contain round line cap")
	}
}

func TestBackendFillRect(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	rect := recording.NewRect(20, 20, 160, 120)
	brush := recording.NewSolidBrush(gg.RGBA{R: 0, G: 1, B: 0, A: 0.78})

	backend.FillRect(rect, brush)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<rect") {
		t.Error("Output should contain rect element")
	}
	if !strings.Contains(svg, `fill-opacity="`) {
		t.Error("Output should contain fill opacity for semi-transparent color")
	}
}

func TestBackendLinearGradient(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	path := gg.NewPath()
	path.Rectangle(50, 50, 200, 150)

	grad := recording.NewLinearGradientBrush(50, 50, 250, 200).
		AddColorStop(0, gg.RGBA{R: 1, G: 0, B: 0, A: 1}).
		AddColorStop(0.5, gg.RGBA{R: 0, G: 1, B: 0, A: 1}).
		AddColorStop(1, gg.RGBA{R: 0, G: 0, B: 1, A: 1})

	backend.FillPath(path, grad, recording.FillRuleNonZero)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<linearGradient") {
		t.Error("Output should contain linearGradient element")
	}
	if !strings.Contains(svg, "<stop") {
		t.Error("Output should contain stop elements")
	}
}

func TestBackendRadialGradient(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	path := gg.NewPath()
	path.Circle(200, 150, 100)

	grad := recording.NewRadialGradientBrush(200, 150, 0, 100).
		AddColorStop(0, gg.RGBA{R: 1, G: 1, B: 0, A: 1}).
		AddColorStop(1, gg.RGBA{R: 1, G: 0, B: 0, A: 1})

	backend.FillPath(path, grad, recording.FillRuleNonZero)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<radialGradient") {
		t.Error("Output should contain radialGradient element")
	}
}

func TestBackendDashedStroke(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	path := gg.NewPath()
	path.MoveTo(50, 150)
	path.LineTo(350, 150)

	brush := recording.NewSolidBrush(gg.RGBA{R: 0, G: 0, B: 0, A: 1})
	stroke := recording.Stroke{
		Width:       3.0,
		Cap:         recording.LineCapButt,
		Join:        recording.LineJoinMiter,
		MiterLimit:  4.0,
		DashPattern: []float64{10, 5, 3, 5},
		DashOffset:  0,
	}

	backend.StrokePath(path, brush, stroke)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `stroke-dasharray="10 5 3 5"`) {
		t.Error("Output should contain dash array")
	}
}

func TestBackendClip(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Create clip path (circle)
	clipPath := gg.NewPath()
	clipPath.Circle(200, 150, 80)

	// Set clip
	backend.SetClip(clipPath, recording.FillRuleNonZero)

	// Draw rectangle (should be clipped to circle)
	rect := gg.NewPath()
	rect.Rectangle(100, 50, 200, 200)

	brush := recording.NewSolidBrush(gg.RGBA{R: 1, G: 0.39, B: 0.39, A: 1})
	backend.FillPath(rect, brush, recording.FillRuleNonZero)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<clipPath") {
		t.Error("Output should contain clipPath element")
	}
	if !strings.Contains(svg, `clip-path="url(#`) {
		t.Error("Output should reference clip path")
	}
}

func TestBackendTransform(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Set transform
	backend.SetTransform(recording.Translate(100, 50))

	path := gg.NewPath()
	path.Rectangle(10, 10, 30, 30)

	brush := recording.NewSolidBrush(gg.RGBA{R: 0.39, G: 0.39, B: 1, A: 1})
	backend.FillPath(path, brush, recording.FillRuleNonZero)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `transform="matrix(`) {
		t.Error("Output should contain transform attribute")
	}
}

func TestBackendSaveToFile(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Draw something
	path := gg.NewPath()
	path.Rectangle(50, 50, 300, 200)
	brush := recording.NewSolidBrush(gg.RGBA{R: 0.39, G: 0.59, B: 0.78, A: 1})
	backend.FillPath(path, brush, recording.FillRuleNonZero)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	// Save to temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.svg")

	err = backend.SaveToFile(filePath)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists and has content
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if info.Size() == 0 {
		t.Error("SVG file should not be empty")
	}

	// Verify SVG header
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.HasPrefix(string(data), "<?xml") {
		t.Error("Output file should start with XML declaration")
	}
	if !strings.Contains(string(data), "<svg") {
		t.Error("Output file should contain SVG element")
	}
}

func TestBackendText(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	brush := recording.NewSolidBrush(gg.RGBA{R: 0, G: 0, B: 0, A: 1})
	backend.DrawText("Hello, SVG!", 100, 150, nil, brush)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<text") {
		t.Error("Output should contain text element")
	}
	if !strings.Contains(svg, "Hello, SVG!") {
		t.Error("Output should contain the text content")
	}
}

func TestBackendTextXMLEscape(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	brush := recording.NewSolidBrush(gg.RGBA{R: 0, G: 0, B: 0, A: 1})
	backend.DrawText("<script>alert('xss')</script>", 100, 150, nil, brush)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if strings.Contains(svg, "<script>") {
		t.Error("Output should escape XML special characters")
	}
	if !strings.Contains(svg, "&lt;script&gt;") {
		t.Error("Output should contain escaped version")
	}
}

func TestBackendFillRuleEvenOdd(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	path := gg.NewPath()
	path.Rectangle(50, 50, 200, 200)
	path.Rectangle(100, 100, 100, 100) // Inner rectangle

	brush := recording.NewSolidBrush(gg.RGBA{R: 1, G: 0, B: 0, A: 1})
	backend.FillPath(path, brush, recording.FillRuleEvenOdd)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `fill-rule="evenodd"`) {
		t.Error("Output should contain evenodd fill rule")
	}
}

func TestPathToD(t *testing.T) {
	backend := NewBackend()

	path := gg.NewPath()
	path.MoveTo(10, 20)
	path.LineTo(30, 40)
	path.QuadraticTo(50, 60, 70, 80)
	path.CubicTo(90, 100, 110, 120, 130, 140)
	path.Close()

	d := backend.pathToD(path)

	if !strings.Contains(d, "M10 20") {
		t.Error("Path data should contain MoveTo command")
	}
	if !strings.Contains(d, "L30 40") {
		t.Error("Path data should contain LineTo command")
	}
	if !strings.Contains(d, "Q50 60 70 80") {
		t.Error("Path data should contain QuadTo command")
	}
	if !strings.Contains(d, "C90 100 110 120 130 140") {
		t.Error("Path data should contain CubicTo command")
	}
	if !strings.Contains(d, "Z") {
		t.Error("Path data should contain Close command")
	}
}

func TestColorToCSS(t *testing.T) {
	tests := []struct {
		color    gg.RGBA
		expected string
	}{
		{gg.RGBA{R: 1, G: 0, B: 0, A: 1}, "rgb(255,0,0)"},
		{gg.RGBA{R: 0, G: 1, B: 0, A: 1}, "rgb(0,255,0)"},
		{gg.RGBA{R: 0, G: 0, B: 1, A: 1}, "rgb(0,0,255)"},
		{gg.RGBA{R: 0.5, G: 0.5, B: 0.5, A: 1}, "rgb(127,127,127)"},
	}

	for _, tt := range tests {
		result := colorToCSS(tt.color)
		if result != tt.expected {
			t.Errorf("colorToCSS(%v) = %s, expected %s", tt.color, result, tt.expected)
		}
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&apos;s"},
	}

	for _, tt := range tests {
		result := escapeXML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeXML(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestSweepGradientFallback(t *testing.T) {
	backend := NewBackend()
	err := backend.Begin(400, 300)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	path := gg.NewPath()
	path.Circle(200, 150, 100)

	// Sweep gradients are not supported in SVG, should fallback to first stop color
	grad := recording.NewSweepGradientBrush(200, 150, 0).
		AddColorStop(0, gg.RGBA{R: 1, G: 0, B: 0, A: 1}).
		AddColorStop(1, gg.RGBA{R: 0, G: 1, B: 0, A: 1})

	backend.FillPath(path, grad, recording.FillRuleNonZero)

	err = backend.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	_, err = backend.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	// Should fallback to first stop color (red)
	if !strings.Contains(svg, `fill="rgb(255,0,0)"`) {
		t.Error("Sweep gradient should fallback to first stop color")
	}
}

func BenchmarkBackendFillPath(b *testing.B) {
	backend := NewBackend()
	_ = backend.Begin(800, 600)

	path := gg.NewPath()
	path.Rectangle(50, 50, 100, 80)
	brush := recording.NewSolidBrush(gg.RGBA{R: 1, G: 0, B: 0, A: 1})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.FillPath(path, brush, recording.FillRuleNonZero)
	}
}

func BenchmarkBackendStrokePath(b *testing.B) {
	backend := NewBackend()
	_ = backend.Begin(800, 600)

	path := gg.NewPath()
	path.MoveTo(0, 0)
	path.LineTo(100, 100)
	path.LineTo(200, 0)

	brush := recording.NewSolidBrush(gg.RGBA{R: 0, G: 0, B: 0, A: 1})
	stroke := recording.DefaultStroke()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend.StrokePath(path, brush, stroke)
	}
}
