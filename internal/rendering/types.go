package rendering

const (
	MIMETypeHTML       = "text/html"
	MIMETypeJavaScript = "application/javascript"
	MIMETypeJPEG       = "image/jpeg"
	MIMETypeJSON       = "application/json"
	MIMETypeLatex      = "text/latex"
	MIMETypeMarkdown   = "text/markdown"
	MIMETypePNG        = "image/png"
	MIMETypePDF        = "application/pdf"
	MIMETypeSVG        = "image/svg+xml"
	MIMETypeText       = "text/plain"
)

type MIMEMap = map[string]interface{}

type Data = struct {
	Data      MIMEMap
	Metadata  MIMEMap
	Transient MIMEMap
}

type Renderer = interface {
	Render() Data
}

type SimpleRenderer = interface {
	SimpleRender() MIMEMap
}

type HTMLer = interface {
	HTML() string
}

type JavaScripter = interface {
	JavaScript() string
}

type JPEGer = interface {
	JPEG() []byte
}

type JSONer = interface {
	JSON() map[string]interface{}
}

type Latexer = interface {
	Latex() string
}

type Markdowner = interface {
	Markdown() string
}

type PNGer = interface {
	PNG() []byte
}

type PDFer = interface {
	PDF() []byte
}

type SVGer = interface {
	SVG() string
}
