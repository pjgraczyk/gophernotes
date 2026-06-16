package kernel

import (
	"fmt"
	"reflect"
	"strings"

	basereflect "github.com/pjgraczyk/gomacro/base/reflect"
	"github.com/pjgraczyk/gomacro/xreflect"

	"github.com/gopherdata/gophernotes/internal/rendering"
)

func (kernel *Kernel) initRenderers() {
	kernel.render = make(map[string]xreflect.Type)
	for name, typ := range kernel.display.Types {
		if typ.Kind() == reflect.Interface {
			kernel.render[name] = typ
		}
	}
}

func (kernel *Kernel) canAutoRender(data interface{}, typ xreflect.Type) bool {
	if rendering.CanAutoRender(data) {
		return true
	}
	if kernel == nil || typ == nil {
		return false
	}
	for _, xtyp := range kernel.render {
		if typ.Implements(xtyp) {
			return true
		}
	}
	return false
}

func (kernel *Kernel) autoRender(mimeType string, arg interface{}, typ xreflect.Type) rendering.Data {
	var data rendering.Data
	if x, ok := arg.(rendering.Data); ok {
		data = x
	}

	if kernel == nil || typ == nil {
		return rendering.AutoRenderAll(mimeType, arg)
	}

	for name, xtyp := range kernel.render {
		fun := rendering.AutoRenderers[name]
		if fun == nil || !typ.Implements(xtyp) {
			continue
		}
		conv := kernel.ir.Comp.Converter(typ, xtyp)
		x := arg
		if conv != nil {
			x = basereflect.ValueInterface(conv(xreflect.ValueOf(x)))
			if x == nil {
				continue
			}
		}
		data = fun(data, x)
	}
	return rendering.FillDefaults(data, arg, "", nil, "", nil)
}

func (kernel *Kernel) autoRenderResults(vals []interface{}, types []xreflect.Type) rendering.Data {
	filtered := kernel.filterResults(vals, types)

	if len(filtered) == 0 {
		return rendering.Data{}
	}

	if len(filtered) == 1 {
		v := filtered[0]
		if kernel.canAutoRender(v.val, v.typ) {
			return kernel.autoRender("", v.val, v.typ)
		}
		return rendering.MakeData(rendering.MIMETypeText, formatValue(v.val, v.typ))
	}

	var buf strings.Builder
	for i, v := range filtered {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(formatValue(v.val, v.typ))
	}
	return rendering.MakeData(rendering.MIMETypeText, buf.String())
}

type resultEntry struct {
	val interface{}
	typ xreflect.Type
}

func (kernel *Kernel) filterResults(vals []interface{}, types []xreflect.Type) []resultEntry {
	var filtered []resultEntry
	hasMultipleReturns := len(vals) > 1

	for i, val := range vals {
		isErr := kernel.isErrorType(types[i])

		if hasMultipleReturns && isErr && val == nil {
			continue
		}

		filtered = append(filtered, resultEntry{val: val, typ: types[i]})
	}

	return filtered
}

func formatValue(val interface{}, typ xreflect.Type) string {
	typeStr := "nil"
	if typ != nil {
		typeStr = typ.String()
	}
	return fmt.Sprintf("%v (%s)", val, typeStr)
}

func (kernel *Kernel) isErrorType(typ xreflect.Type) bool {
	if typ == nil {
		return false
	}
	return typ.String() == "error"
}
