package rendering

import (
	"image"
	"image/color"
	r "reflect"

	"github.com/pjgraczyk/gomacro/imports"
)

var Display = imports.Package{
	Binds: map[string]r.Value{
		"Any":                r.ValueOf(Any),
		"Auto":               r.ValueOf(Auto),
		"File":               r.ValueOf(File),
		"HTML":               r.ValueOf(HTML),
		"Image":              r.ValueOf(Image),
		"JPEG":               r.ValueOf(JPEG),
		"JSON":               r.ValueOf(JSON),
		"JavaScript":         r.ValueOf(JavaScript),
		"Latex":              r.ValueOf(Latex),
		"MakeData":           r.ValueOf(MakeData),
		"MakeData3":          r.ValueOf(MakeData3),
		"Markdown":           r.ValueOf(Markdown),
		"Math":               r.ValueOf(Math),
		"MIME":               r.ValueOf(MIME),
		"MIMETypeHTML":       r.ValueOf(MIMETypeHTML),
		"MIMETypeJavaScript": r.ValueOf(MIMETypeJavaScript),
		"MIMETypeJPEG":       r.ValueOf(MIMETypeJPEG),
		"MIMETypeJSON":       r.ValueOf(MIMETypeJSON),
		"MIMETypeLatex":      r.ValueOf(MIMETypeLatex),
		"MIMETypeMarkdown":   r.ValueOf(MIMETypeMarkdown),
		"MIMETypePDF":        r.ValueOf(MIMETypePDF),
		"MIMETypePNG":        r.ValueOf(MIMETypePNG),
		"MIMETypeSVG":        r.ValueOf(MIMETypeSVG),
		"PDF":                r.ValueOf(PDF),
		"PNG":                r.ValueOf(PNG),
		"SVG":                r.ValueOf(SVG),
	},
	Types: map[string]r.Type{
		"Data":           r.TypeOf((*Data)(nil)).Elem(),
		"HTMLer":         r.TypeOf((*HTMLer)(nil)).Elem(),
		"JavaScripter":   r.TypeOf((*JavaScripter)(nil)).Elem(),
		"Image":          r.TypeOf((*image.Image)(nil)).Elem(),
		"JPEGer":         r.TypeOf((*JPEGer)(nil)).Elem(),
		"JSONer":         r.TypeOf((*JSONer)(nil)).Elem(),
		"Latexer":        r.TypeOf((*Latexer)(nil)).Elem(),
		"Markdowner":     r.TypeOf((*Markdowner)(nil)).Elem(),
		"MIMEMap":        r.TypeOf((*MIMEMap)(nil)).Elem(),
		"PNGer":          r.TypeOf((*PNGer)(nil)).Elem(),
		"PDFer":          r.TypeOf((*PDFer)(nil)).Elem(),
		"Renderer":       r.TypeOf((*Renderer)(nil)).Elem(),
		"SimpleRenderer": r.TypeOf((*SimpleRenderer)(nil)).Elem(),
		"SVGer":          r.TypeOf((*SVGer)(nil)).Elem(),
	},
	Proxies: map[string]r.Type{
		"HTMLer":         r.TypeOf((*proxyHTMLer)(nil)).Elem(),
		"Image":          r.TypeOf((*proxyImageImage)(nil)).Elem(),
		"JPEGer":         r.TypeOf((*proxyJPEGer)(nil)).Elem(),
		"JSONer":         r.TypeOf((*proxyJSONer)(nil)).Elem(),
		"Latexer":        r.TypeOf((*proxyLatexer)(nil)).Elem(),
		"Markdowner":     r.TypeOf((*proxyMarkdowner)(nil)).Elem(),
		"PNGer":          r.TypeOf((*proxyPNGer)(nil)).Elem(),
		"PDFer":          r.TypeOf((*proxyPDFer)(nil)).Elem(),
		"Renderer":       r.TypeOf((*proxyRenderer)(nil)).Elem(),
		"SimpleRenderer": r.TypeOf((*proxySimpleRenderer)(nil)).Elem(),
		"SVGer":          r.TypeOf((*proxySVGer)(nil)).Elem(),
	},
}

type proxyHTMLer struct {
	Object interface{}
	HTML_  func(interface{}) string
}

func (p *proxyHTMLer) HTML() string {
	return p.HTML_(p.Object)
}

var _ HTMLer = (*proxyHTMLer)(nil)

type proxyJPEGer struct {
	Object interface{}
	JPEG_  func(interface{}) []byte
}

func (p *proxyJPEGer) JPEG() []byte {
	return p.JPEG_(p.Object)
}

var _ JPEGer = (*proxyJPEGer)(nil)

type proxyJSONer struct {
	Object interface{}
	JSON_  func(interface{}) map[string]interface{}
}

func (p *proxyJSONer) JSON() map[string]interface{} {
	return p.JSON_(p.Object)
}

var _ JSONer = (*proxyJSONer)(nil)

type proxyLatexer struct {
	Object interface{}
	Latex_ func(interface{}) string
}

func (p *proxyLatexer) Latex() string {
	return p.Latex_(p.Object)
}

var _ Latexer = (*proxyLatexer)(nil)

type proxyMarkdowner struct {
	Object    interface{}
	Markdown_ func(interface{}) string
}

func (p *proxyMarkdowner) Markdown() string {
	return p.Markdown_(p.Object)
}

var _ Markdowner = (*proxyMarkdowner)(nil)

type proxyPNGer struct {
	Object interface{}
	PNG_   func(interface{}) []byte
}

func (p *proxyPNGer) PNG() []byte {
	return p.PNG_(p.Object)
}

var _ PNGer = (*proxyPNGer)(nil)

type proxyPDFer struct {
	Object interface{}
	PDF_   func(interface{}) []byte
}

func (p *proxyPDFer) PDF() []byte {
	return p.PDF_(p.Object)
}

var _ PDFer = (*proxyPDFer)(nil)

type proxyRenderer struct {
	Object  interface{}
	Render_ func(interface{}) Data
}

func (p *proxyRenderer) Render() Data {
	return p.Render_(p.Object)
}

var _ Renderer = (*proxyRenderer)(nil)

type proxySimpleRenderer struct {
	Object        interface{}
	SimpleRender_ func(interface{}) MIMEMap
}

func (p *proxySimpleRenderer) SimpleRender() MIMEMap {
	return p.SimpleRender_(p.Object)
}

var _ SimpleRenderer = (*proxySimpleRenderer)(nil)

type proxySVGer struct {
	Object interface{}
	SVG_   func(interface{}) string
}

func (p *proxySVGer) SVG() string {
	return p.SVG_(p.Object)
}

var _ SVGer = (*proxySVGer)(nil)

type proxyImageImage struct {
	Object      interface{}
	At_         func(interface{}, int, int) color.Color
	Bounds_     func(interface{}) image.Rectangle
	ColorModel_ func(interface{}) color.Model
}

func (p *proxyImageImage) At(x int, y int) color.Color {
	return p.At_(p.Object, x, y)
}

func (p *proxyImageImage) Bounds() image.Rectangle {
	return p.Bounds_(p.Object)
}

func (p *proxyImageImage) ColorModel() color.Model {
	return p.ColorModel_(p.Object)
}

var _ image.Image = (*proxyImageImage)(nil)

func init() {
	imports.Packages["display"] = Display
	imports.Packages["github.com/gopherdata/gophernotes"] = Display
}
