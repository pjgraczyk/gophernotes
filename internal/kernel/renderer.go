package kernel

import (
	"reflect"

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
	var nilcount int
	var obj interface{}
	var typ xreflect.Type
	for i, val := range vals {
		if kernel.canAutoRender(val, types[i]) {
			obj = val
			typ = types[i]
		} else if val == nil {
			nilcount++
		}
	}
	if obj != nil && nilcount == len(vals)-1 {
		return kernel.autoRender("", obj, typ)
	}
	if nilcount == len(vals) {
		return rendering.Data{}
	}
	return rendering.MakeData(rendering.MIMETypeText, rendering.AnyToString(vals...))
}
