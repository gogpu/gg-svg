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
	"image/draw"
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

	// emitTransforms is enabled for direct Backend use. Recorder geometry is
	// already in world space, so the registry constructs a playback backend
	// with this disabled to avoid applying Recorder transforms a second time.
	emitTransforms bool

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
	currentClipIDs   []string
}

// backendState stores the graphics state for Save/Restore operations.
type backendState struct {
	transform recording.Matrix
	clipIDs   []string
}

// NewBackend creates a new SVG backend.
// The backend starts in an uninitialized state. Call Begin() to initialize
// with specific dimensions before drawing.
func NewBackend() *Backend {
	return &Backend{
		emitTransforms: true,
		stateStack:     make([]backendState, 0, 8),
	}
}

// newPlaybackBackend creates the backend registered with recording. Recorder
// operations eagerly transform paths, rectangles, images, text positions, and
// clips into world coordinates, despite Playback also forwarding transform
// state. Suppressing SVG transform attributes at this integration seam keeps
// that world-space geometry unchanged while NewBackend retains transform
// support for callers that drive the Backend interface directly.
func newPlaybackBackend() *Backend {
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
	b.currentClipIDs = b.currentClipIDs[:0]

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
		clipIDs:   append([]string(nil), b.currentClipIDs...),
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
	b.currentClipIDs = state.clipIDs

	if b.groupDepth > 0 {
		b.builder.WriteString("</g>")
		b.groupDepth--
	}
}

// SetTransform sets the current transformation matrix.
func (b *Backend) SetTransform(m recording.Matrix) {
	b.currentTransform = m
}

// SetClip intersects the current clipping region with the given path.
func (b *Backend) SetClip(path *gg.Path, rule recording.FillRule) {
	if path == nil {
		return
	}

	clipID := b.nextID("clip")
	b.currentClipIDs = append(b.currentClipIDs, clipID)

	// Capture the clip-time transform in the definition. Draw elements use
	// their own transforms inside untransformed clip wrapper groups.
	fmt.Fprintf(&b.defs, `<clipPath id="%s" clipPathUnits="userSpaceOnUse">`, clipID)
	b.defs.WriteString(`<path`)
	b.writeTransformTo(&b.defs)
	fmt.Fprintf(&b.defs, ` d="%s"`, b.pathToD(path))
	if rule == recording.FillRuleEvenOdd {
		b.defs.WriteString(` clip-rule="evenodd"`)
	}
	b.defs.WriteString(`/></clipPath>`)
}

// ClearClip removes any clipping region.
func (b *Backend) ClearClip() {
	b.currentClipIDs = b.currentClipIDs[:0]
}

// FillPath fills the given path with the brush color/pattern.
func (b *Backend) FillPath(path *gg.Path, brush recording.Brush, rule recording.FillRule) {
	if path == nil {
		return
	}

	b.openClipGroups()
	b.builder.WriteString("<path")
	b.writeTransform()
	fmt.Fprintf(&b.builder, ` d="%s"`, b.pathToD(path))
	b.writeFill(brush)
	if rule == recording.FillRuleEvenOdd {
		b.builder.WriteString(` fill-rule="evenodd"`)
	}
	b.builder.WriteString(` stroke="none"`)
	b.builder.WriteString("/>")
	b.closeClipGroups()
}

// StrokePath strokes the given path with the brush and stroke style.
func (b *Backend) StrokePath(path *gg.Path, brush recording.Brush, stroke recording.Stroke) {
	if path == nil {
		return
	}

	b.openClipGroups()
	b.builder.WriteString("<path")
	b.writeTransform()
	fmt.Fprintf(&b.builder, ` d="%s"`, b.pathToD(path))
	b.builder.WriteString(` fill="none"`)
	b.writeStroke(brush, stroke)
	b.builder.WriteString("/>")
	b.closeClipGroups()
}

// FillRect fills an axis-aligned rectangle with the brush.
func (b *Backend) FillRect(rect recording.Rect, brush recording.Brush) {
	b.openClipGroups()
	b.builder.WriteString("<rect")
	b.writeTransform()
	fmt.Fprintf(&b.builder, ` x="%g" y="%g" width="%g" height="%g"`,
		rect.MinX, rect.MinY, rect.Width(), rect.Height())
	b.writeFill(brush)
	b.builder.WriteString(` stroke="none"`)
	b.builder.WriteString("/>")
	b.closeClipGroups()
}

// DrawImage draws an image from the source rectangle to the destination rectangle.
func (b *Backend) DrawImage(img image.Image, src, dst recording.Rect, opts recording.ImageOptions) {
	if img == nil {
		return
	}

	// A zero source rectangle means the entire image. Source rectangles that
	// extend past the image bounds are clipped and their visible portion is
	// mapped to the corresponding portion of the destination rectangle.
	img, dst, ok := cropImage(img, src, dst)
	if !ok {
		return
	}

	// Encode image to PNG and then to base64 data URI
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return
	}
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	b.openClipGroups()
	b.builder.WriteString("<image")
	b.writeTransform()
	fmt.Fprintf(&b.builder, ` x="%g" y="%g" width="%g" height="%g"`,
		dst.MinX, dst.MinY, dst.Width(), dst.Height())
	fmt.Fprintf(&b.builder, ` href="%s"`, dataURI)

	if opts.Alpha < 1.0 {
		fmt.Fprintf(&b.builder, ` opacity="%g"`, opts.Alpha)
	}

	b.builder.WriteString(` preserveAspectRatio="none"`)
	b.builder.WriteString("/>")
	b.closeClipGroups()
}

// cropImage returns an image containing the source rectangle and the
// destination rectangle where that image should be rendered. Source
// coordinates are local to the image (the same zero-origin coordinates used
// by recording.Recorder), even when img.Bounds has a non-zero minimum. Pixel
// boundaries are rounded outwards so a partially covered edge pixel is not
// silently dropped.
func cropImage(img image.Image, src, dst recording.Rect) (image.Image, recording.Rect, bool) {
	if dst.Width() <= 0 || dst.Height() <= 0 {
		return nil, recording.Rect{}, false
	}

	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil, recording.Rect{}, false
	}

	// SrcRect is optional in a DrawImageCommand. Treat either zero dimension
	// as the documented full-image sentinel, while rejecting negative sizes.
	if src.Width() == 0 || src.Height() == 0 {
		src = recording.NewRect(
			0,
			0,
			float64(bounds.Dx()),
			float64(bounds.Dy()),
		)
	}
	if src.Width() <= 0 || src.Height() <= 0 {
		return nil, recording.Rect{}, false
	}

	// Translate the recording-local source rectangle into image coordinates
	// before clipping and copying pixels.
	src = recording.NewRect(
		src.MinX+float64(bounds.Min.X),
		src.MinY+float64(bounds.Min.Y),
		src.Width(),
		src.Height(),
	)

	// Clip in source coordinates before converting to integer pixel bounds.
	minX := math.Max(src.MinX, float64(bounds.Min.X))
	minY := math.Max(src.MinY, float64(bounds.Min.Y))
	maxX := math.Min(src.MaxX, float64(bounds.Max.X))
	maxY := math.Min(src.MaxY, float64(bounds.Max.Y))
	if minX >= maxX || minY >= maxY {
		return nil, recording.Rect{}, false
	}

	// image.Image pixels are addressed by integer coordinates. Round outwards
	// so clipping at an image edge never produces an empty crop for a source
	// rectangle that intersects a pixel.
	cropMinX := maxInt(bounds.Min.X, int(math.Floor(minX)))
	cropMinY := maxInt(bounds.Min.Y, int(math.Floor(minY)))
	cropMaxX := minInt(bounds.Max.X, int(math.Ceil(maxX)))
	cropMaxY := minInt(bounds.Max.Y, int(math.Ceil(maxY)))
	if cropMinX >= cropMaxX || cropMinY >= cropMaxY {
		return nil, recording.Rect{}, false
	}

	// Keep the source's non-premultiplied channels when copying transparent
	// pixels. Using image.RGBA here would round/unpremultiply NRGBA values.
	cropped := image.NewNRGBA(image.Rect(0, 0, cropMaxX-cropMinX, cropMaxY-cropMinY))
	draw.Draw(cropped, cropped.Bounds(), img, image.Pt(cropMinX, cropMinY), draw.Src)

	// Keep source-to-destination scaling intact when the source rectangle was
	// clipped at an image edge. Use the clipped floating-point source bounds
	// rather than the outward-rounded crop bounds so fractional source
	// rectangles do not expand the destination unexpectedly.
	srcW := src.Width()
	srcH := src.Height()
	dstW := dst.Width()
	dstH := dst.Height()
	dstMinX, dstMinY := dst.MinX, dst.MinY
	dst.MaxX = dstMinX + (maxX-src.MinX)*dstW/srcW
	dst.MinX = dstMinX + (minX-src.MinX)*dstW/srcW
	dst.MaxY = dstMinY + (maxY-src.MinY)*dstH/srcH
	dst.MinY = dstMinY + (minY-src.MinY)*dstH/srcH

	return cropped, dst, true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DrawText draws text at the given position with the specified font face and brush.
func (b *Backend) DrawText(s string, x, y float64, face text.Face, brush recording.Brush) {
	b.openClipGroups()
	b.builder.WriteString("<text")
	b.writeTransform()
	fmt.Fprintf(&b.builder, ` x="%g" y="%g"`, x, y)

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
	fmt.Fprintf(&b.builder, ` font-size="%g"`, fontSize)

	// Fill color
	b.writeFill(brush)

	b.builder.WriteString(">")
	b.builder.WriteString(escapeXML(s))
	b.builder.WriteString("</text>")
	b.closeClipGroups()
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
			fmt.Fprintf(&d, "M%g %g", e.Point.X, e.Point.Y)
		case gg.LineTo:
			fmt.Fprintf(&d, "L%g %g", e.Point.X, e.Point.Y)
		case gg.QuadTo:
			fmt.Fprintf(&d, "Q%g %g %g %g",
				e.Control.X, e.Control.Y, e.Point.X, e.Point.Y)
		case gg.CubicTo:
			fmt.Fprintf(&d, "C%g %g %g %g %g %g",
				e.Control1.X, e.Control1.Y,
				e.Control2.X, e.Control2.Y,
				e.Point.X, e.Point.Y)
		case gg.Close:
			d.WriteString("Z")
		}
	}

	return d.String()
}

// writeTransform writes the transform attribute if not identity.
func (b *Backend) writeTransform() {
	b.writeTransformTo(&b.builder)
}

// writeTransformTo writes the current transform to dst when this backend
// receives local-space geometry. Registered recorder playback supplies
// world-space geometry and disables transform emission.
func (b *Backend) writeTransformTo(dst *strings.Builder) {
	if !b.emitTransforms {
		return
	}
	m := b.currentTransform
	if m.IsIdentity() {
		return
	}
	fmt.Fprintf(dst, ` transform="matrix(%g,%g,%g,%g,%g,%g)"`,
		m.A, m.D, m.B, m.E, m.C, m.F)
}

// openClipGroups opens one group per active clip. Nested clipped groups are
// used instead of chained clipPath definitions because some SVG renderers do
// not implement clip-path on clipPath elements.
func (b *Backend) openClipGroups() {
	for _, clipID := range b.currentClipIDs {
		fmt.Fprintf(&b.builder, `<g clip-path="url(#%s)">`, clipID)
	}
}

// closeClipGroups closes groups opened by openClipGroups.
func (b *Backend) closeClipGroups() {
	for range b.currentClipIDs {
		b.builder.WriteString(`</g>`)
	}
}

// writeFill writes fill attributes for a brush.
func (b *Backend) writeFill(brush recording.Brush) {
	switch br := brush.(type) {
	case recording.SolidBrush:
		fmt.Fprintf(&b.builder, ` fill="%s"`, colorToCSS(br.Color))
		if br.Color.A < 1.0 {
			fmt.Fprintf(&b.builder, ` fill-opacity="%g"`, br.Color.A)
		}

	case *recording.LinearGradientBrush:
		gradID := b.addLinearGradient(br)
		fmt.Fprintf(&b.builder, ` fill="url(#%s)"`, gradID)

	case *recording.RadialGradientBrush:
		gradID := b.addRadialGradient(br)
		fmt.Fprintf(&b.builder, ` fill="url(#%s)"`, gradID)

	case *recording.SweepGradientBrush:
		// SVG doesn't support sweep gradients directly
		// Fallback to first stop color
		if len(br.Stops) > 0 {
			fmt.Fprintf(&b.builder, ` fill="%s"`, colorToCSS(br.Stops[0].Color))
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
		fmt.Fprintf(&b.builder, ` stroke="%s"`, colorToCSS(br.Color))
		if br.Color.A < 1.0 {
			fmt.Fprintf(&b.builder, ` stroke-opacity="%g"`, br.Color.A)
		}

	case *recording.LinearGradientBrush:
		gradID := b.addLinearGradient(br)
		fmt.Fprintf(&b.builder, ` stroke="url(#%s)"`, gradID)

	case *recording.RadialGradientBrush:
		gradID := b.addRadialGradient(br)
		fmt.Fprintf(&b.builder, ` stroke="url(#%s)"`, gradID)

	default:
		b.builder.WriteString(` stroke="black"`)
	}

	// Stroke width
	fmt.Fprintf(&b.builder, ` stroke-width="%g"`, stroke.Width)

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
			fmt.Fprintf(&b.builder, ` stroke-miterlimit="%g"`, stroke.MiterLimit)
		}
	}

	// Dash pattern
	if len(stroke.DashPattern) > 0 {
		dashStrs := make([]string, len(stroke.DashPattern))
		for i, v := range stroke.DashPattern {
			dashStrs[i] = fmt.Sprintf("%g", v)
		}
		fmt.Fprintf(&b.builder, ` stroke-dasharray="%s"`, strings.Join(dashStrs, " "))
		if stroke.DashOffset != 0 {
			fmt.Fprintf(&b.builder, ` stroke-dashoffset="%g"`, stroke.DashOffset)
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
	fmt.Fprintf(&b.defs,
		`<linearGradient id="%s" gradientUnits="userSpaceOnUse" x1="%g" y1="%g" x2="%g" y2="%g">`,
		gradID, br.Start.X, br.Start.Y, br.End.X, br.End.Y)

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
		fmt.Fprintf(&b.defs,
			`<stop offset="%g" stop-color="%s"`,
			stop.Offset, colorToCSS(stop.Color))
		if stop.Color.A < 1.0 {
			fmt.Fprintf(&b.defs, ` stop-opacity="%g"`, stop.Color.A)
		}
		b.defs.WriteString(`/>`)
	}

	b.defs.WriteString(`</linearGradient>`)
	return gradID
}

// addRadialGradient adds a radial gradient definition and returns its ID.
func (b *Backend) addRadialGradient(br *recording.RadialGradientBrush) string {
	gradID := b.nextID("rg")

	fmt.Fprintf(&b.defs,
		`<radialGradient id="%s" gradientUnits="userSpaceOnUse" cx="%g" cy="%g" r="%g" fx="%g" fy="%g">`,
		gradID, br.Center.X, br.Center.Y, br.EndRadius, br.Focus.X, br.Focus.Y)

	// Handle spread mode
	switch br.Extend {
	case recording.ExtendRepeat:
		b.defs.WriteString(` spreadMethod="repeat"`)
	case recording.ExtendReflect:
		b.defs.WriteString(` spreadMethod="reflect"`)
	}

	for _, stop := range br.Stops {
		fmt.Fprintf(&b.defs,
			`<stop offset="%g" stop-color="%s"`,
			stop.Offset, colorToCSS(stop.Color))
		if stop.Color.A < 1.0 {
			fmt.Fprintf(&b.defs, ` stop-opacity="%g"`, stop.Color.A)
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
