package rendering

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"strings"
)

func StubDisplay(Data) error {
	return errors.New("cannot display: connection with Jupyter not available")
}

func CanAutoRender(data interface{}) bool {
	switch data.(type) {
	case Data, Renderer, SimpleRenderer, HTMLer, JavaScripter, JPEGer, JSONer,
		Latexer, Markdowner, PNGer, PDFer, SVGer, image.Image:
		return true
	}
	return false
}

var AutoRenderers = map[string]func(Data, interface{}) Data{
	"Renderer": func(d Data, i interface{}) Data {
		if r, ok := i.(Renderer); ok {
			x := r.Render()
			d.Data = Merge(d.Data, x.Data)
			d.Metadata = Merge(d.Metadata, x.Metadata)
			d.Transient = Merge(d.Transient, x.Transient)
		}
		return d
	},
	"SimpleRenderer": func(d Data, i interface{}) Data {
		if r, ok := i.(SimpleRenderer); ok {
			x := r.SimpleRender()
			d.Data = Merge(d.Data, x)
		}
		return d
	},
	"HTMLer": func(d Data, i interface{}) Data {
		if r, ok := i.(HTMLer); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypeHTML] = r.HTML()
		}
		return d
	},
	"JavaScripter": func(d Data, i interface{}) Data {
		if r, ok := i.(JavaScripter); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypeJavaScript] = r.JavaScript()
		}
		return d
	},
	"JPEGer": func(d Data, i interface{}) Data {
		if r, ok := i.(JPEGer); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypeJPEG] = r.JPEG()
		}
		return d
	},
	"JSONer": func(d Data, i interface{}) Data {
		if r, ok := i.(JSONer); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypeJSON] = r.JSON()
		}
		return d
	},
	"Latexer": func(d Data, i interface{}) Data {
		if r, ok := i.(Latexer); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypeLatex] = r.Latex()
		}
		return d
	},
	"Markdowner": func(d Data, i interface{}) Data {
		if r, ok := i.(Markdowner); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypeMarkdown] = r.Markdown()
		}
		return d
	},
	"PNGer": func(d Data, i interface{}) Data {
		if r, ok := i.(PNGer); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypePNG] = r.PNG()
		}
		return d
	},
	"PDFer": func(d Data, i interface{}) Data {
		if r, ok := i.(PDFer); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypePDF] = r.PDF()
		}
		return d
	},
	"SVGer": func(d Data, i interface{}) Data {
		if r, ok := i.(SVGer); ok {
			d.Data = Ensure(d.Data)
			d.Data[MIMETypeSVG] = r.SVG()
		}
		return d
	},
	"Image": func(d Data, i interface{}) Data {
		if r, ok := i.(image.Image); ok {
			b, mimeType, err := EncodePng(r)
			if err != nil {
				d = MakeDataErr(err)
			} else {
				d.Data = Ensure(d.Data)
				d.Data[mimeType] = b
				d.Metadata = Merge(d.Metadata, ImageMetadata(r))
			}
		}
		return d
	},
}

func AutoRenderAll(mimeType string, arg interface{}) Data {
	var data Data
	if x, ok := arg.(Data); ok {
		data = x
	}
	for _, fun := range AutoRenderers {
		data = fun(data, arg)
	}
	return FillDefaults(data, arg, "", nil, "", nil)
}

func Render(mimeType string, data interface{}) Data {
	if CanAutoRender(data) {
		return AutoRenderAll(mimeType, data)
	}
	var s string
	var b []byte
	var err error
	switch d := data.(type) {
	case string:
		s = d
	case []byte:
		b = d
	case io.Reader:
		b, err = io.ReadAll(d)
	case io.WriterTo:
		var buf bytes.Buffer
		d.WriteTo(&buf)
		b = buf.Bytes()
	default:
		panic(fmt.Errorf("unsupported type, cannot render: %T", data))
	}
	return FillDefaults(Data{}, data, s, b, mimeType, err)
}

func Any(mimeType string, data interface{}) Data {
	return Render(mimeType, data)
}

func Auto(data interface{}) Data {
	return Render("", data)
}

func MakeData(mimeType string, data interface{}) Data {
	d := Data{
		Data: MIMEMap{
			mimeType: data,
		},
	}
	if mimeType != MIMETypeText {
		d.Data[MIMETypeText] = fmt.Sprint(data)
	}
	return d
}

func MakeData3(mimeType string, plaintext string, data interface{}) Data {
	return Data{
		Data: MIMEMap{
			MIMETypeText: plaintext,
			mimeType:     data,
		},
	}
}

func MIME(data, metadata MIMEMap) Data {
	return Data{data, metadata, nil}
}

func HTML(html string) Data {
	return MakeData(MIMETypeHTML, html)
}

func JavaScript(javascript string) Data {
	return MakeData(MIMETypeJavaScript, javascript)
}

func JPEG(jpeg []byte) Data {
	return MakeData(MIMETypeJPEG, jpeg)
}

func JSON(json map[string]interface{}) Data {
	return MakeData(MIMETypeJSON, json)
}

func Latex(latex string) Data {
	return MakeData3(MIMETypeLatex, latex, "$"+strings.Trim(latex, "$")+"$")
}

func Markdown(markdown string) Data {
	return MakeData(MIMETypeMarkdown, markdown)
}

func Math(latex string) Data {
	return MakeData3(MIMETypeLatex, latex, "$$"+strings.Trim(latex, "$")+"$$")
}

func PDF(pdf []byte) Data {
	return MakeData(MIMETypePDF, pdf)
}

func PNG(png []byte) Data {
	return MakeData(MIMETypePNG, png)
}

func SVG(svg string) Data {
	return MakeData(MIMETypeSVG, svg)
}

func File(mimeType string, path string) Data {
	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return Any(mimeType, bytes)
}

func Ensure(bundle MIMEMap) MIMEMap {
	if bundle == nil {
		bundle = make(MIMEMap)
	}
	return bundle
}

func Merge(a MIMEMap, b MIMEMap) MIMEMap {
	if len(b) == 0 {
		return a
	}
	if a == nil {
		a = make(MIMEMap)
	}
	for k, v := range b {
		a[k] = v
	}
	return a
}

func FillDefaults(data Data, arg interface{}, s string, b []byte, mimeType string, err error) Data {
	if err != nil {
		return MakeDataErr(err)
	}
	if data.Data == nil {
		data.Data = make(MIMEMap)
	}
	if len(s) != 0 && len(mimeType) != 0 {
		data.Data[mimeType] = s
	}
	if data.Data[MIMETypeText] == "" {
		if len(s) == 0 {
			s = fmt.Sprint(arg)
		}
		data.Data[MIMETypeText] = s
	}
	if len(b) != 0 {
		if len(mimeType) == 0 {
			mimeType = http.DetectContentType(b)
		}
		if len(mimeType) != 0 && mimeType != MIMETypeText {
			data.Data[mimeType] = b
		}
	}
	return data
}

func MakeDataErr(err error) Data {
	return Data{
		Data: MIMEMap{
			"ename":     "ERROR",
			"evalue":    err.Error(),
			"traceback": nil,
			"status":    "error",
		},
	}
}

func AnyToString(vals ...interface{}) string {
	var buf strings.Builder
	for i, val := range vals {
		if i != 0 {
			buf.WriteByte(' ')
		}
		fmt.Fprint(&buf, val)
	}
	return buf.String()
}
