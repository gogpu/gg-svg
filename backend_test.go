package svg

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"math"
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
	if !strings.Contains(svg, `<clipPath id="clip1" clipPathUnits="userSpaceOnUse">`) {
		t.Error("Output should contain clipPath element")
	}
	if !strings.Contains(svg, `clip-path="url(#`) {
		t.Error("Output should reference clip path")
	}
}

func TestBackendClipIntersectionCapturesTransforms(t *testing.T) {
	backend := NewBackend()
	if err := backend.Begin(100, 100); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	first := gg.NewPath()
	first.Rectangle(0, 0, 60, 100)
	backend.SetTransform(recording.Translate(10, 0))
	backend.SetClip(first, recording.FillRuleNonZero)

	second := gg.NewPath()
	second.Rectangle(0, 0, 100, 60)
	backend.SetTransform(recording.Translate(0, 20))
	backend.SetClip(second, recording.FillRuleEvenOdd)

	backend.SetTransform(recording.Translate(5, 5))
	backend.FillRect(
		recording.NewRect(0, 0, 90, 90),
		recording.NewSolidBrush(gg.RGBA{R: 1, A: 1}),
	)

	svg := backendSVG(t, backend)
	if !strings.Contains(svg, `<clipPath id="clip1" clipPathUnits="userSpaceOnUse"><path transform="matrix(1,0,0,1,10,0)" d="M0 0L60 0L60 100L0 100Z"/></clipPath>`) {
		t.Errorf("first clip did not capture its transform:\n%s", svg)
	}
	if !strings.Contains(svg, `<clipPath id="clip2" clipPathUnits="userSpaceOnUse"><path transform="matrix(1,0,0,1,0,20)" d="M0 0L100 0L100 60L0 60Z" clip-rule="evenodd"/></clipPath>`) {
		t.Errorf("second clip did not capture its transform and fill rule:\n%s", svg)
	}
	want := `<g clip-path="url(#clip1)"><g clip-path="url(#clip2)"><rect transform="matrix(1,0,0,1,5,5)" x="0" y="0" width="90" height="90" fill="rgb(255,0,0)" stroke="none"/></g></g>`
	if !strings.Contains(svg, want) {
		t.Errorf("successive clips were not emitted as an intersection; want %q in:\n%s", want, svg)
	}
	requireValidSVG(t, svg)
}

func TestBackendClipNilAndEmptyPaths(t *testing.T) {
	backend := NewBackend()
	if err := backend.Begin(20, 20); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	backend.SetClip(nil, recording.FillRuleNonZero)
	backend.SetClip(gg.NewPath(), recording.FillRuleNonZero)
	backend.FillRect(
		recording.NewRect(0, 0, 20, 20),
		recording.NewSolidBrush(gg.RGBA{R: 1, A: 1}),
	)

	svg := backendSVG(t, backend)
	if got := strings.Count(svg, `<clipPath id=`); got != 1 {
		t.Fatalf("nil/empty paths created %d clip definitions, want one empty clip:\n%s", got, svg)
	}
	if !strings.Contains(svg, `<clipPath id="clip1" clipPathUnits="userSpaceOnUse"><path d=""/></clipPath>`) {
		t.Errorf("empty path did not create an empty clipping region:\n%s", svg)
	}
	if !strings.Contains(svg, `<g clip-path="url(#clip1)"><rect`) {
		t.Errorf("empty clipping region was not applied to subsequent drawing:\n%s", svg)
	}
	requireValidSVG(t, svg)
}

func TestBackendClipSaveRestoreAndClear(t *testing.T) {
	backend := NewBackend()
	if err := backend.Begin(100, 100); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	clip := func(x float64) {
		path := gg.NewPath()
		path.Rectangle(x, 0, 20, 100)
		backend.SetClip(path, recording.FillRuleNonZero)
	}
	red := recording.NewSolidBrush(gg.RGBA{R: 1, A: 1})
	blue := recording.NewSolidBrush(gg.RGBA{B: 1, A: 1})
	green := recording.NewSolidBrush(gg.RGBA{G: 1, A: 1})
	rect := recording.NewRect(0, 0, 100, 100)

	clip(0)  // clip1
	clip(10) // clip2
	backend.Save()
	backend.ClearClip()
	clip(20) // clip3; must not overwrite the saved clip slice
	backend.FillRect(rect, red)
	backend.Restore()
	backend.FillRect(rect, blue)
	backend.ClearClip()
	backend.FillRect(rect, green)

	svg := backendSVG(t, backend)
	insideSave := `<g><g clip-path="url(#clip3)"><rect x="0" y="0" width="100" height="100" fill="rgb(255,0,0)" stroke="none"/></g></g>`
	if !strings.Contains(svg, insideSave) {
		t.Errorf("ClearClip did not replace the active saved-scope clips:\n%s", svg)
	}
	restored := `<g clip-path="url(#clip1)"><g clip-path="url(#clip2)"><rect x="0" y="0" width="100" height="100" fill="rgb(0,0,255)" stroke="none"/></g></g>`
	if !strings.Contains(svg, restored) {
		t.Errorf("Restore did not recover the complete clip intersection:\n%s", svg)
	}
	unclipped := `<rect x="0" y="0" width="100" height="100" fill="rgb(0,255,0)" stroke="none"/>`
	if !strings.Contains(svg, restored+unclipped) {
		t.Errorf("ClearClip did not remove every active clip after Restore:\n%s", svg)
	}
	requireValidSVG(t, svg)
}

func TestBackendClipAppliesToEveryDrawable(t *testing.T) {
	backend := NewBackend()
	if err := backend.Begin(100, 100); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	first := gg.NewPath()
	first.Rectangle(0, 0, 80, 100)
	backend.SetClip(first, recording.FillRuleNonZero)
	second := gg.NewPath()
	second.Rectangle(0, 0, 100, 80)
	backend.SetClip(second, recording.FillRuleNonZero)

	path := gg.NewPath()
	path.Rectangle(0, 0, 100, 100)
	brush := recording.NewSolidBrush(gg.RGBA{R: 1, A: 1})
	backend.FillPath(path, brush, recording.FillRuleNonZero)
	backend.StrokePath(path, brush, recording.DefaultStroke())
	backend.FillRect(recording.NewRect(0, 0, 100, 100), brush)
	backend.DrawImage(
		image.NewRGBA(image.Rect(0, 0, 1, 1)),
		recording.NewRect(0, 0, 1, 1),
		recording.NewRect(0, 0, 100, 100),
		recording.DefaultImageOptions(),
	)
	backend.DrawText("clipped <&", 0, 20, nil, brush)

	svg := backendSVG(t, backend)
	prefix := `<g clip-path="url(#clip1)"><g clip-path="url(#clip2)">`
	if got := strings.Count(svg, prefix); got != 5 {
		t.Fatalf("clip intersection wrapped %d drawable types, want 5:\n%s", got, svg)
	}
	for _, element := range []string{"<path", "<rect", "<image", "<text"} {
		if !strings.Contains(svg, prefix+element) {
			t.Errorf("%s output was not clipped by the full intersection:\n%s", element, svg)
		}
	}
	if !strings.Contains(svg, `>clipped &lt;&amp;</text>`) {
		t.Errorf("clipped text content was not XML escaped:\n%s", svg)
	}
	requireValidSVG(t, svg)
}

func TestRecordingPlaybackClipUsesWorldSpace(t *testing.T) {
	recorder := recording.NewRecorder(100, 100)
	recorder.Translate(10, 0)
	recorder.DrawRectangle(0, 0, 60, 100)
	recorder.Clip()
	recorder.Identity()
	recorder.Translate(0, 20)
	recorder.DrawRectangle(0, 0, 100, 60)
	recorder.Clip()
	recorder.Identity()
	recorder.SetFillRGBA(1, 0, 0, 1)
	recorder.FillRectangle(0, 0, 100, 100)

	backend, err := recording.NewBackend("svg")
	if err != nil {
		t.Fatalf("NewBackend failed: %v", err)
	}
	if err := recorder.FinishRecording().Playback(backend); err != nil {
		t.Fatalf("Playback failed: %v", err)
	}
	writer, ok := backend.(recording.WriterBackend)
	if !ok {
		t.Fatal("registered SVG backend does not implement recording.WriterBackend")
	}
	var buf bytes.Buffer
	if _, err := writer.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	svg := buf.String()
	if strings.Contains(svg, ` transform="matrix(`) {
		t.Errorf("recorder world-space clips or geometry were transformed twice:\n%s", svg)
	}
	if !strings.Contains(svg, `<clipPath id="clip1" clipPathUnits="userSpaceOnUse"><path d="M10 0L70 0L70 100L10 100Z"/></clipPath>`) {
		t.Errorf("first recorder clip was not emitted in world space:\n%s", svg)
	}
	if !strings.Contains(svg, `<clipPath id="clip2" clipPathUnits="userSpaceOnUse"><path d="M0 20L100 20L100 80L0 80Z"/></clipPath>`) {
		t.Errorf("second recorder clip was not emitted in world space:\n%s", svg)
	}
	want := `<g clip-path="url(#clip1)"><g clip-path="url(#clip2)"><rect x="0" y="0" width="100" height="100" fill="rgb(255,0,0)" stroke="none"/></g></g>`
	if !strings.Contains(svg, want) {
		t.Errorf("recorder clips were not intersected:\n%s", svg)
	}
	requireValidSVG(t, svg)
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
	if !strings.Contains(svg, `transform="matrix(1,0,0,1,100,50)" d="M10 10L40 10L40 40L10 40Z"`) {
		t.Errorf("Direct backend output should apply its transform to local geometry:\n%s", svg)
	}
}

func TestBackendTransformSVGMatrixOrder(t *testing.T) {
	tests := []struct {
		name      string
		transform recording.Matrix
		want      string
	}{
		{
			name:      "asymmetric shear",
			transform: recording.Shear(2, 3),
			want:      `transform="matrix(1,3,2,1,0,0)"`,
		},
		{
			name: "quarter turn",
			transform: recording.Matrix{
				A: 0, B: -1, C: 0,
				D: 1, E: 0, F: 0,
			},
			want: `transform="matrix(0,1,-1,0,0,0)"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := NewBackend()
			if err := backend.Begin(100, 100); err != nil {
				t.Fatalf("Begin failed: %v", err)
			}
			backend.SetTransform(tt.transform)

			path := gg.NewPath()
			path.Rectangle(10, 10, 20, 20)
			backend.FillPath(path, recording.NewSolidBrush(gg.RGBA{R: 1, A: 1}), recording.FillRuleNonZero)

			var buf bytes.Buffer
			if _, err := backend.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo failed: %v", err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Errorf("SVG transform has incorrect coefficient order; want %q in:\n%s", tt.want, buf.String())
			}
		})
	}
}

func TestRecordingPlaybackDoesNotApplyTransformTwice(t *testing.T) {
	recorder := recording.NewRecorder(200, 150)
	recorder.Translate(10, 20)
	recorder.Scale(2, 3)
	recorder.SetFillRGBA(1, 0, 0, 1)
	recorder.DrawRectangle(1, 2, 4, 5)
	recorder.Fill()
	recorder.FillRectangle(3, 4, 5, 6)

	backend, err := recording.NewBackend("svg")
	if err != nil {
		t.Fatalf("NewBackend failed: %v", err)
	}
	if err := recorder.FinishRecording().Playback(backend); err != nil {
		t.Fatalf("Playback failed: %v", err)
	}

	var buf bytes.Buffer
	writer, ok := backend.(recording.WriterBackend)
	if !ok {
		t.Fatal("registered SVG backend does not implement recording.WriterBackend")
	}
	if _, err := writer.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, ` transform="matrix(`) {
		t.Errorf("Playback emitted a transform for world-space recorder geometry:\n%s", output)
	}
	if !strings.Contains(output, `d="M12 26L20 26L20 41L12 41Z"`) {
		t.Errorf("Playback path does not use the recorder's world-space coordinates:\n%s", output)
	}
	if !strings.Contains(output, `<rect x="16" y="32" width="10" height="18"`) {
		t.Errorf("Playback rectangle does not use the recorder's world-space coordinates:\n%s", output)
	}
}

func TestBackendDrawImageCropsSource(t *testing.T) {
	img := distinctColorImage()
	backend := NewBackend()
	if err := backend.Begin(40, 30); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	src := recording.NewRect(1, 1, 2, 2)
	dst := recording.NewRect(5, 6, 20, 10)
	backend.DrawImage(img, src, dst, recording.DefaultImageOptions())
	if err := backend.End(); err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := backend.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	embedded, element := decodeEmbeddedImage(t, buf.String())
	if got := embedded.Bounds().Size(); got != (image.Point{X: 2, Y: 2}) {
		t.Fatalf("embedded image size = %v, want 2x2", got)
	}

	want := [][]color.NRGBA{
		{{255, 128, 0, 255}, {128, 0, 255, 255}},
		{{64, 64, 64, 255}, {192, 192, 192, 255}},
	}
	for y, row := range want {
		for x, wantPixel := range row {
			gotPixel := color.NRGBAModel.Convert(embedded.At(x, y)).(color.NRGBA)
			if gotPixel != wantPixel {
				t.Errorf("embedded pixel (%d,%d) = %v, want %v", x, y, gotPixel, wantPixel)
			}
		}
	}

	if !strings.Contains(element, `x="5" y="6" width="20" height="10"`) {
		t.Errorf("destination rectangle missing or changed: %s", element)
	}
}

func TestBackendDrawImageUsesLocalCoordinatesForNonZeroImageBounds(t *testing.T) {
	img := distinctColorImageAt(image.Rect(10, 20, 14, 24))
	backend := NewBackend()
	if err := backend.Begin(40, 30); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// Recorder source rectangles are local to the image, regardless of the
	// image.Image bounds origin.
	backend.DrawImage(img, recording.NewRect(1, 1, 2, 2), recording.NewRect(5, 6, 20, 10), recording.DefaultImageOptions())
	if err := backend.End(); err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := backend.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	embedded, _ := decodeEmbeddedImage(t, buf.String())
	if got := embedded.Bounds().Size(); got != (image.Point{X: 2, Y: 2}) {
		t.Fatalf("embedded image size = %v, want 2x2", got)
	}
	want := [][]color.NRGBA{
		{{255, 128, 0, 255}, {128, 0, 255, 255}},
		{{64, 64, 64, 255}, {192, 192, 192, 255}},
	}
	for y, row := range want {
		for x, wantPixel := range row {
			gotPixel := color.NRGBAModel.Convert(embedded.At(x, y)).(color.NRGBA)
			if gotPixel != wantPixel {
				t.Errorf("embedded pixel (%d,%d) = %v, want %v", x, y, gotPixel, wantPixel)
			}
		}
	}
}

func TestBackendDrawImageFractionalSourcePreservesDestinationMapping(t *testing.T) {
	backend := NewBackend()
	if err := backend.Begin(40, 30); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// The crop image needs integer pixel bounds, but the SVG destination must
	// still represent the requested fractional source rectangle exactly.
	backend.DrawImage(distinctColorImage(), recording.NewRect(0.25, 0.25, 1.5, 1.5), recording.NewRect(5, 6, 20, 10), recording.DefaultImageOptions())
	if err := backend.End(); err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := backend.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	_, element := decodeEmbeddedImage(t, buf.String())
	if !strings.Contains(element, `x="5" y="6" width="20" height="10"`) {
		t.Errorf("fractional source changed destination mapping: %s", element)
	}
}

func TestBackendDrawImagePreservesUnpremultipliedColors(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 64, B: 32, A: 128})
	img.SetNRGBA(2, 0, color.NRGBA{R: 12, G: 34, B: 56, A: 0})

	backend := NewBackend()
	if err := backend.Begin(40, 30); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	backend.DrawImage(img, recording.NewRect(1, 0, 2, 1), recording.NewRect(5, 6, 20, 10), recording.DefaultImageOptions())
	if err := backend.End(); err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := backend.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	embedded, _ := decodeEmbeddedImage(t, buf.String())
	want := []color.NRGBA{
		{R: 255, G: 64, B: 32, A: 128},
		{R: 12, G: 34, B: 56, A: 0},
	}
	for x, wantPixel := range want {
		gotPixel := color.NRGBAModel.Convert(embedded.At(x, 0)).(color.NRGBA)
		if gotPixel != wantPixel {
			t.Errorf("embedded pixel (%d,0) = %v, want %v", x, gotPixel, wantPixel)
		}
	}
}

func TestBackendDrawImageClipsSourceAtImageEdge(t *testing.T) {
	img := distinctColorImage()
	backend := NewBackend()
	if err := backend.Begin(40, 30); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	// The left source pixel lies outside img. The visible two-thirds should
	// occupy the corresponding two-thirds of the destination rectangle.
	src := recording.NewRect(-1, 1, 3, 2)
	dst := recording.NewRect(4, 5, 9, 8)
	backend.DrawImage(img, src, dst, recording.DefaultImageOptions())
	if err := backend.End(); err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := backend.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	embedded, element := decodeEmbeddedImage(t, buf.String())
	if got := embedded.Bounds().Size(); got != (image.Point{X: 2, Y: 2}) {
		t.Fatalf("embedded image size = %v, want 2x2", got)
	}
	if !strings.Contains(element, `x="7" y="5" width="6" height="8"`) {
		t.Errorf("clipped destination rectangle missing or changed: %s", element)
	}
}

func TestBackendDrawImageUsesFullImageForZeroSourceDimension(t *testing.T) {
	tests := []struct {
		name string
		src  recording.Rect
	}{
		{name: "zero width", src: recording.NewRect(1, 1, 0, 2)},
		{name: "zero height", src: recording.NewRect(1, 1, 2, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg := drawImageSVG(
				t,
				distinctColorImage(),
				tt.src,
				recording.NewRect(5, 6, 20, 10),
			)
			embedded, element := decodeEmbeddedImage(t, svg)
			if got := embedded.Bounds().Size(); got != (image.Point{X: 4, Y: 4}) {
				t.Fatalf("embedded image size = %v, want 4x4", got)
			}
			if !strings.Contains(element, `x="5" y="6" width="20" height="10"`) {
				t.Errorf("destination rectangle missing or changed: %s", element)
			}
		})
	}
}

func TestBackendDrawImageOmitsInvalidRegions(t *testing.T) {
	tests := []struct {
		name string
		img  image.Image
		src  recording.Rect
		dst  recording.Rect
	}{
		{
			name: "zero destination width",
			img:  distinctColorImage(),
			src:  recording.NewRect(0, 0, 1, 1),
			dst:  recording.NewRect(5, 6, 0, 10),
		},
		{
			name: "empty image",
			img:  image.NewRGBA(image.Rect(0, 0, 0, 4)),
			src:  recording.NewRect(0, 0, 1, 1),
			dst:  recording.NewRect(5, 6, 20, 10),
		},
		{
			name: "negative source width",
			img:  distinctColorImage(),
			src:  recording.NewRect(1, 1, -1, 2),
			dst:  recording.NewRect(5, 6, 20, 10),
		},
		{
			name: "source outside image",
			img:  distinctColorImage(),
			src:  recording.NewRect(10, 10, 2, 2),
			dst:  recording.NewRect(5, 6, 20, 10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg := drawImageSVG(t, tt.img, tt.src, tt.dst)
			if strings.Contains(svg, "<image") {
				t.Errorf("invalid image region emitted an SVG image element: %s", svg)
			}
		})
	}
}

func TestCropImageRejectsNaNSource(t *testing.T) {
	src := recording.Rect{MinX: math.NaN(), MinY: 0, MaxX: math.NaN(), MaxY: 1}
	if _, _, ok := cropImage(distinctColorImage(), src, recording.NewRect(0, 0, 1, 1)); ok {
		t.Fatal("cropImage accepted a source rectangle with NaN coordinates")
	}
}

func TestCropImageIntegerBoundsHelpers(t *testing.T) {
	if got := minInt(1, 2); got != 1 {
		t.Errorf("minInt(1, 2) = %d, want 1", got)
	}
	if got := maxInt(2, 1); got != 2 {
		t.Errorf("maxInt(2, 1) = %d, want 2", got)
	}
}

func drawImageSVG(t *testing.T, img image.Image, src, dst recording.Rect) string {
	t.Helper()
	backend := NewBackend()
	if err := backend.Begin(40, 30); err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	backend.DrawImage(img, src, dst, recording.DefaultImageOptions())
	if err := backend.End(); err != nil {
		t.Fatalf("End failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := backend.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	return buf.String()
}

func distinctColorImage() *image.RGBA {
	return distinctColorImageAt(image.Rect(0, 0, 4, 4))
}

func distinctColorImageAt(bounds image.Rectangle) *image.RGBA {
	img := image.NewRGBA(bounds)
	colors := []color.RGBA{
		{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 0, 255, 255},
		{255, 255, 0, 255}, {255, 128, 0, 255}, {128, 0, 255, 255}, {0, 255, 255, 255},
		{128, 128, 128, 255}, {64, 64, 64, 255}, {192, 192, 192, 255}, {255, 255, 255, 255},
		{0, 0, 0, 255}, {64, 128, 192, 255}, {192, 128, 64, 255}, {32, 96, 160, 255},
	}
	for i, c := range colors {
		img.SetRGBA(bounds.Min.X+i%4, bounds.Min.Y+i/4, c)
	}
	return img
}

func decodeEmbeddedImage(t *testing.T, svg string) (image.Image, string) {
	t.Helper()
	start := strings.Index(svg, "<image")
	if start < 0 {
		t.Fatalf("SVG output does not contain an image element: %s", svg)
	}
	relEnd := strings.Index(svg[start:], "/>")
	if relEnd < 0 {
		t.Fatalf("SVG image element is not self-closing: %s", svg[start:])
	}
	element := svg[start : start+relEnd+2]
	const prefix = `href="data:image/png;base64,`
	hrefStart := strings.Index(element, prefix)
	if hrefStart < 0 {
		t.Fatalf("SVG image element has no PNG data URI: %s", element)
	}
	hrefStart += len(prefix)
	hrefEnd := strings.IndexByte(element[hrefStart:], '"')
	if hrefEnd < 0 {
		t.Fatalf("SVG image data URI is unterminated: %s", element)
	}
	pngBytes, err := base64.StdEncoding.DecodeString(element[hrefStart : hrefStart+hrefEnd])
	if err != nil {
		t.Fatalf("decode embedded PNG: %v", err)
	}
	embedded, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode embedded PNG image: %v", err)
	}
	return embedded, element
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

func backendSVG(t *testing.T, backend *Backend) string {
	t.Helper()
	if err := backend.End(); err != nil {
		t.Fatalf("End failed: %v", err)
	}
	var buf bytes.Buffer
	if _, err := backend.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	return buf.String()
}

func requireValidSVG(t *testing.T, svg string) {
	t.Helper()
	var document struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal([]byte(svg), &document); err != nil {
		t.Fatalf("output is not well-formed XML: %v\n%s", err, svg)
	}
	if document.XMLName.Local != "svg" {
		t.Fatalf("output root is %q, want svg", document.XMLName.Local)
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
