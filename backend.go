// Package svg provides an SVG export backend for gg's recording system.
//
// This package registers an "svg" backend that can be used to export
// recorded drawing operations to SVG format.
//
// # Usage
//
// Import this package with a blank identifier to register the SVG backend:
//
//	import _ "github.com/gogpu/gg-svg"
//
// Then use recording.NewBackend("svg") to create an SVG backend:
//
//	backend, err := recording.NewBackend("svg")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Record drawing operations
//	rec := recording.NewRecorder(800, 600)
//	// ... draw ...
//	r := rec.Finish()
//
//	// Playback to SVG backend
//	r.Playback(backend)
//
//	// Save to file
//	if fb, ok := backend.(recording.FileBackend); ok {
//	    fb.SaveToFile("output.svg")
//	}
package svg

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"strings"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/recording"
	"github.com/gogpu/gg/text"
)

// Backend implements recording.Backend for SVG output.
// It generates SVG XML from recorded drawing commands.
type Backend struct {
	width  int
	height int

	// SVG content builder
	builder strings.Builder

	// Definitions (gradients, clip paths)
	defs strings.Builder

	// Current group nesting for Save/Restore
	groupDepth int

	// Counter for unique IDs
	idCounter int

	// State stack for Save/Restore
	stateStack []backendState

	// Current graphics state
	currentTransform recording.Matrix
	currentClipID    string
}

// backendState stores the graphics state for Save/Restore operations.
type backendState struct {
	transform recording.Matrix
	clipID    string
}

// NewBackend creates a new SVG backend.
// The backend starts in an uninitialized state. Call Begin() to initialize
// with specific dimensions before drawing.
func NewBackend() *Backend {
	return &Backend{
		stateStack: make([]backendState, 0, 8),
	}
}

// Begin initializes the backend for rendering at the given dimensions.
func (b *Backend) Begin(width, height int) error {
	b.width = width
	b.height = height
	b.builder.Reset()
	b.defs.Reset()
	b.groupDepth = 0
	b.idCounter = 0
	b.stateStack = b.stateStack[:0]
	b.currentTransform = recording.Identity()
	b.currentClipID = ""

	return nil
}

// End finalizes the rendering.
func (b *Backend) End() error {
	return nil
}

// Save saves the current graphics state onto a stack.
func (b *Backend) Save() {
	b.stateStack = append(b.stateStack, backendState{
		transform: b.currentTransform,
		clipID:    b.currentClipID,
	})
	b.builder.WriteString("<g>")
	b.groupDepth++
}

// Restore restores the graphics state from the stack.
func (b *Backend) Restore() {
	if len(b.stateStack) == 0 {
		return
	}

	state := b.stateStack[len(b.stateStack)-1]
	b.stateStack = b.stateStack[:len(b.stateStack)-1]

	b.currentTransform = state.transform
	b.currentClipID = state.clipID

	if b.groupDepth > 0 {
		b.builder.WriteString("</g>")
		b.groupDepth--
	}
}

// SetTransform sets the current transformation matrix.
func (b *Backend) SetTransform(m recording.Matrix) {
	b.currentTransform = m
}

// SetClip sets the clipping region to the given path.
func (b *Backend) SetClip(path *gg.Path, rule recording.FillRule) {
	if path == nil {
		return
	}

	clipID := b.nextID("clip")
	b.currentClipID = clipID

	// Write clip path definition
	b.defs.WriteString(fmt.Sprintf(`<clipPath id="%s">`, clipID))
	b.defs.WriteString(fmt.Sprintf(`<path d="%s"`, b.pathToD(path)))
	if rule == recording.FillRuleEvenOdd {
		b.defs.WriteString(` clip-rule="evenodd"`)
	}
	b.defs.WriteString(`/></clipPath>`)
}

// ClearClip removes any clipping region.
func (b *Backend) ClearClip() {
	b.currentClipID = ""
}

// FillPath fills the given path with the brush color/pattern.
func (b *Backend) FillPath(path *gg.Path, brush recording.Brush, rule recording.FillRule) {
	if path == nil {
		return
	}

	b.builder.WriteString("<path")
	b.writeTransform()
	b.writeClip()
	b.builder.WriteString(fmt.Sprintf(` d="%s"`, b.pathToD(path)))
	b.writeFill(brush)
	if rule == recording.FillRuleEvenOdd {
		b.builder.WriteString(` fill-rule="evenodd"`)
	}
	b.builder.WriteString(` stroke="none"`)
	b.builder.WriteString("/>")
}

// StrokePath strokes the given path with the brush and stroke style.
func (b *Backend) StrokePath(path *gg.Path, brush recording.Brush, stroke recording.Stroke) {
	if path == nil {
		return
	}

	b.builder.WriteString("<path")
	b.writeTransform()
	b.writeClip()
	b.builder.WriteString(fmt.Sprintf(` d="%s"`, b.pathToD(path)))
	b.builder.WriteString(` fill="none"`)
	b.writeStroke(brush, stroke)
	b.builder.WriteString("/>")
}

// FillRect fills an axis-aligned rectangle with the brush.
func (b *Backend) FillRect(rect recording.Rect, brush recording.Brush) {
	b.builder.WriteString("<rect")
	b.writeTransform()
	b.writeClip()
	b.builder.WriteString(fmt.Sprintf(` x="%g" y="%g" width="%g" height="%g"`,
		rect.MinX, rect.MinY, rect.Width(), rect.Height()))
	b.writeFill(brush)
	b.builder.WriteString(` stroke="none"`)
	b.builder.WriteString("/>")
}

// DrawImage draws an image from the source rectangle to the destination rectangle.
func (b *Backend) DrawImage(img image.Image, src, dst recording.Rect, opts recording.ImageOptions) {
	if img == nil {
		return
	}

	// Encode image to PNG and then to base64 data URI
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return
	}
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	b.builder.WriteString("<image")
	b.writeTransform()
	b.writeClip()
	b.builder.WriteString(fmt.Sprintf(` x="%g" y="%g" width="%g" height="%g"`,
		dst.MinX, dst.MinY, dst.Width(), dst.Height()))
	b.builder.WriteString(fmt.Sprintf(` href="%s"`, dataURI))

	if opts.Alpha < 1.0 {
		b.builder.WriteString(fmt.Sprintf(` opacity="%g"`, opts.Alpha))
	}

	b.builder.WriteString(` preserveAspectRatio="none"`)
	b.builder.WriteString("/>")
}

// DrawText draws text at the given position with the specified font face and brush.
func (b *Backend) DrawText(s string, x, y float64, face text.Face, brush recording.Brush) {
	b.builder.WriteString("<text")
	b.writeTransform()
	b.writeClip()
	b.builder.WriteString(fmt.Sprintf(` x="%g" y="%g"`, x, y))

	// Font settings
	fontSize := 12.0
	if face != nil {
		fontSize = face.Size()
		if fontSize <= 0 {
			metrics := face.Metrics()
			fontSize = metrics.LineHeight()
			if fontSize <= 0 {
				fontSize = 12.0
			}
		}
	}
	b.builder.WriteString(fmt.Sprintf(` font-size="%g"`, fontSize))

	// Fill color
	b.writeFill(brush)

	b.builder.WriteString(">")
	b.builder.WriteString(escapeXML(s))
	b.builder.WriteString("</text>")
}

// WriteTo writes the SVG to the given writer.
// This implements recording.WriterBackend.
func (b *Backend) WriteTo(w io.Writer) (int64, error) {
	var total int64

	// Write SVG header
	header := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" viewBox="0 0 %d %d">
`, b.width, b.height, b.width, b.height)
	n, err := w.Write([]byte(header))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Write definitions if any
	defs := b.defs.String()
	if defs != "" {
		n, err = w.Write([]byte("<defs>"))
		total += int64(n)
		if err != nil {
			return total, err
		}
		n, err = w.Write([]byte(defs))
		total += int64(n)
		if err != nil {
			return total, err
		}
		n, err = w.Write([]byte("</defs>\n"))
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// Write content
	n, err = w.Write([]byte(b.builder.String()))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Close any unclosed groups
	for i := 0; i < b.groupDepth; i++ {
		n, err = w.Write([]byte("</g>"))
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	// Write SVG footer
	n, err = w.Write([]byte("\n</svg>\n"))
	total += int64(n)
	return total, err
}

// SaveToFile saves the SVG to a file at the given path.
// This implements recording.FileBackend.
func (b *Backend) SaveToFile(path string) error {
	f, err := os.Create(path) //nolint:gosec // Path is provided by user code
	if err != nil {
		return err
	}

	_, writeErr := b.WriteTo(f)
	closeErr := f.Close()

	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// nextID generates a unique ID for SVG elements.
func (b *Backend) nextID(prefix string) string {
	b.idCounter++
	return fmt.Sprintf("%s%d", prefix, b.idCounter)
}

// pathToD converts a gg.Path to an SVG path data string.
func (b *Backend) pathToD(path *gg.Path) string {
	var d strings.Builder

	for _, elem := range path.Elements() {
		switch e := elem.(type) {
		case gg.MoveTo:
			d.WriteString(fmt.Sprintf("M%g %g", e.Point.X, e.Point.Y))
		case gg.LineTo:
			d.WriteString(fmt.Sprintf("L%g %g", e.Point.X, e.Point.Y))
		case gg.QuadTo:
			d.WriteString(fmt.Sprintf("Q%g %g %g %g",
				e.Control.X, e.Control.Y, e.Point.X, e.Point.Y))
		case gg.CubicTo:
			d.WriteString(fmt.Sprintf("C%g %g %g %g %g %g",
				e.Control1.X, e.Control1.Y,
				e.Control2.X, e.Control2.Y,
				e.Point.X, e.Point.Y))
		case gg.Close:
			d.WriteString("Z")
		}
	}

	return d.String()
}

// writeTransform writes the transform attribute if not identity.
func (b *Backend) writeTransform() {
	m := b.currentTransform
	if m.IsIdentity() {
		return
	}
	b.builder.WriteString(fmt.Sprintf(` transform="matrix(%g,%g,%g,%g,%g,%g)"`,
		m.A, m.B, m.D, m.E, m.C, m.F))
}

// writeClip writes the clip-path attribute if set.
func (b *Backend) writeClip() {
	if b.currentClipID != "" {
		b.builder.WriteString(fmt.Sprintf(` clip-path="url(#%s)"`, b.currentClipID))
	}
}

// writeFill writes fill attributes for a brush.
func (b *Backend) writeFill(brush recording.Brush) {
	switch br := brush.(type) {
	case recording.SolidBrush:
		b.builder.WriteString(fmt.Sprintf(` fill="%s"`, colorToCSS(br.Color)))
		if br.Color.A < 1.0 {
			b.builder.WriteString(fmt.Sprintf(` fill-opacity="%g"`, br.Color.A))
		}

	case *recording.LinearGradientBrush:
		gradID := b.addLinearGradient(br)
		b.builder.WriteString(fmt.Sprintf(` fill="url(#%s)"`, gradID))

	case *recording.RadialGradientBrush:
		gradID := b.addRadialGradient(br)
		b.builder.WriteString(fmt.Sprintf(` fill="url(#%s)"`, gradID))

	case *recording.SweepGradientBrush:
		// SVG doesn't support sweep gradients directly
		// Fallback to first stop color
		if len(br.Stops) > 0 {
			b.builder.WriteString(fmt.Sprintf(` fill="%s"`, colorToCSS(br.Stops[0].Color)))
		} else {
			b.builder.WriteString(` fill="black"`)
		}

	default:
		b.builder.WriteString(` fill="black"`)
	}
}

// writeStroke writes stroke attributes.
func (b *Backend) writeStroke(brush recording.Brush, stroke recording.Stroke) {
	// Stroke color
	switch br := brush.(type) {
	case recording.SolidBrush:
		b.builder.WriteString(fmt.Sprintf(` stroke="%s"`, colorToCSS(br.Color)))
		if br.Color.A < 1.0 {
			b.builder.WriteString(fmt.Sprintf(` stroke-opacity="%g"`, br.Color.A))
		}

	case *recording.LinearGradientBrush:
		gradID := b.addLinearGradient(br)
		b.builder.WriteString(fmt.Sprintf(` stroke="url(#%s)"`, gradID))

	case *recording.RadialGradientBrush:
		gradID := b.addRadialGradient(br)
		b.builder.WriteString(fmt.Sprintf(` stroke="url(#%s)"`, gradID))

	default:
		b.builder.WriteString(` stroke="black"`)
	}

	// Stroke width
	b.builder.WriteString(fmt.Sprintf(` stroke-width="%g"`, stroke.Width))

	// Line cap
	switch stroke.Cap {
	case recording.LineCapRound:
		b.builder.WriteString(` stroke-linecap="round"`)
	case recording.LineCapSquare:
		b.builder.WriteString(` stroke-linecap="square"`)
	default:
		b.builder.WriteString(` stroke-linecap="butt"`)
	}

	// Line join
	switch stroke.Join {
	case recording.LineJoinRound:
		b.builder.WriteString(` stroke-linejoin="round"`)
	case recording.LineJoinBevel:
		b.builder.WriteString(` stroke-linejoin="bevel"`)
	default:
		b.builder.WriteString(` stroke-linejoin="miter"`)
		if stroke.MiterLimit > 0 {
			b.builder.WriteString(fmt.Sprintf(` stroke-miterlimit="%g"`, stroke.MiterLimit))
		}
	}

	// Dash pattern
	if len(stroke.DashPattern) > 0 {
		dashStrs := make([]string, len(stroke.DashPattern))
		for i, v := range stroke.DashPattern {
			dashStrs[i] = fmt.Sprintf("%g", v)
		}
		b.builder.WriteString(fmt.Sprintf(` stroke-dasharray="%s"`, strings.Join(dashStrs, " ")))
		if stroke.DashOffset != 0 {
			b.builder.WriteString(fmt.Sprintf(` stroke-dashoffset="%g"`, stroke.DashOffset))
		}
	}
}

// addLinearGradient adds a linear gradient definition and returns its ID.
func (b *Backend) addLinearGradient(br *recording.LinearGradientBrush) string {
	gradID := b.nextID("lg")

	// Calculate gradient vector
	dx := br.End.X - br.Start.X
	dy := br.End.Y - br.Start.Y
	length := math.Sqrt(dx*dx + dy*dy)

	// Use userSpaceOnUse for absolute coordinates
	b.defs.WriteString(fmt.Sprintf(
		`<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%g" y1="%g" x2="%g" y2="%g">`,
		gradID, br.Start.X, br.Start.Y, br.End.X, br.End.Y))

	// Handle spread mode
	if length > 0 {
		switch br.Extend {
		case recording.ExtendRepeat:
			b.defs.WriteString(` spreadMethod="repeat"`)
		case recording.ExtendReflect:
			b.defs.WriteString(` spreadMethod="reflect"`)
		}
	}

	for _, stop := range br.Stops {
		b.defs.WriteString(fmt.Sprintf(
			`<stop offset="%g" stop-color="%s"`,
			stop.Offset, colorToCSS(stop.Color)))
		if stop.Color.A < 1.0 {
			b.defs.WriteString(fmt.Sprintf(` stop-opacity="%g"`, stop.Color.A))
		}
		b.defs.WriteString(`/>`)
	}

	b.defs.WriteString(`</linearGradient>`)
	return gradID
}

// addRadialGradient adds a radial gradient definition and returns its ID.
func (b *Backend) addRadialGradient(br *recording.RadialGradientBrush) string {
	gradID := b.nextID("rg")

	b.defs.WriteString(fmt.Sprintf(
		`<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%g" cy="%g" r="%g" fx="%g" fy="%g">`,
		gradID, br.Center.X, br.Center.Y, br.EndRadius, br.Focus.X, br.Focus.Y))

	// Handle spread mode
	switch br.Extend {
	case recording.ExtendRepeat:
		b.defs.WriteString(` spreadMethod="repeat"`)
	case recording.ExtendReflect:
		b.defs.WriteString(` spreadMethod="reflect"`)
	}

	for _, stop := range br.Stops {
		b.defs.WriteString(fmt.Sprintf(
			`<stop offset="%g" stop-color="%s"`,
			stop.Offset, colorToCSS(stop.Color)))
		if stop.Color.A < 1.0 {
			b.defs.WriteString(fmt.Sprintf(` stop-opacity="%g"`, stop.Color.A))
		}
		b.defs.WriteString(`/>`)
	}

	b.defs.WriteString(`</radialGradient>`)
	return gradID
}

// colorToCSS converts an RGBA color to CSS color string.
// gg.RGBA uses float64 values in the range [0, 1].
func colorToCSS(c gg.RGBA) string {
	r := int(c.R * 255)
	g := int(c.G * 255)
	b := int(c.B * 255)
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}

// escapeXML escapes special XML characters.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// Ensure Backend implements the required interfaces.
var (
	_ recording.Backend       = (*Backend)(nil)
	_ recording.WriterBackend = (*Backend)(nil)
	_ recording.FileBackend   = (*Backend)(nil)
)
