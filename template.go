package knife

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"text/template"

	"github.com/gostaticanalysis/analysisutil"
	"github.com/gostaticanalysis/comment"
)

type TempalteData struct {
	Fset      *token.FileSet
	Files     []*ast.File
	TypesInfo *types.Info
	Pkg       *types.Package
	Extra     map[string]interface{}
}

// NewTemplate creates new a template with funcmap.
func NewTemplate(td *TempalteData) *template.Template {
	prefix := td.Pkg.Name()
	return template.New(prefix + "_format").Funcs(newFuncMap(td))
}

func newFuncMap(td *TempalteData) template.FuncMap {
	var cmaps comment.Maps
	return template.FuncMap{
		"pkg":        func() *Package { return NewPackage(td.Pkg) },
		"br":         fmt.Sprintln,
		"array":      ToArray,
		"basic":      ToBasic,
		"chan":       ToChan,
		"interface":  ToInterface,
		"map":        ToMap,
		"named":      ToNamed,
		"pointer":    ToPointer,
		"ptr":        ToPointer,
		"signature":  ToSignature,
		"slice":      ToSlice,
		"struct":     ToStruct,
		"len":        func(v interface{}) int { return reflect.ValueOf(v).Len() },
		"cap":        func(v interface{}) int { return reflect.ValueOf(v).Cap() },
		"last":       td.last,
		"exported":   Exported,
		"methods":    Methods,
		"names":      td.names,
		"implements": implements,
		"identical":  identical,
		"under":      under,
		"pos":        func(v interface{}) token.Position { return Position(td.Fset, v) },
		"objectof":   func(s string) Object { return td.objectOf(s) },
		"typeof":     func(s string) *Type { return td.typeOf(s) },
		"doc":        func(v interface{}) string { return td.doc(cmaps, v) },
		"data":       func(k string) interface{} { return td.Extra[k] },
	}
}

func (td *TempalteData) names(slice interface{}) string {
	vs := reflect.ValueOf(slice)
	switch vs.Kind() {
	case reflect.Slice, reflect.Array:
		return td.nameSlice(vs)
	case reflect.Map:
		return td.nameMap(vs)
	}

	return ""
}

func (td *TempalteData) nameSlice(vs reflect.Value) string {
	var buf bytes.Buffer
	for i := 0; i < vs.Len(); i++ {
		s := td.name(vs.Index(i))
		if s != "" {
			fmt.Fprintln(&buf, s)
		}
	}
	return strings.TrimRight(buf.String(), "\n")
}

func (td *TempalteData) nameMap(vs reflect.Value) string {
	var buf bytes.Buffer
	for _, k := range vs.MapKeys() {
		s := td.name(vs.MapIndex(k))
		if s != "" {
			fmt.Fprintln(&buf, s)
		}
	}
	return strings.TrimRight(buf.String(), "\n")
}

func (td *TempalteData) name(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Ptr:
		return td.name(v.Elem())
	case reflect.Struct:
		fv := v.FieldByName("Name")
		if !fv.IsZero() {
			return fv.String()
		}
	}
	return ""
}

func (td *TempalteData) objectOf(s string) Object {
	dotPos := strings.LastIndex(s, ".")

	if dotPos == -1 {
		obj := types.Universe.Lookup(s)
		return NewObject(obj)
	} else {
		pkg, name := s[:dotPos], s[dotPos+1:]
		obj := analysisutil.LookupFromImports(td.Pkg.Imports(), pkg, name)
		if obj != nil {
			return NewObject(obj)
		}
		if analysisutil.RemoveVendor(td.Pkg.Name()) != analysisutil.RemoveVendor(pkg) {
			return nil
		}
		return NewObject(td.Pkg.Scope().Lookup(name))
	}
}

func (td *TempalteData) typeOf(s string) *Type {
	if s == "" {
		return nil
	}

	if s[0] == '*' {
		typ := td.typeOf(s[1:])
		if typ == nil {
			return nil
		}
		return NewType(types.NewPointer(typ.TypesType))
	}

	obj := td.objectOf(s)
	if obj == nil {
		return nil
	}
	return NewType(obj.TypesObject().Type())
}

func (td *TempalteData) doc(cmaps comment.Maps, v interface{}) string {
	node, ok := v.(interface{ Pos() token.Pos })
	if !ok {
		return ""
	}

	if cmaps == nil {
		cmaps = comment.New(td.Fset, td.Files)
	}

	pos := node.Pos()
	cgs := cmaps.CommentsByPosLine(td.Fset, pos)
	if len(cgs) > 0 {
		return strings.TrimSpace(cgs[len(cgs)-1].Text())
	}

	return ""
}

func (td *TempalteData) last(v interface{}) interface{} {
	_v := reflect.ValueOf(v)
	return _v.Index(_v.Len() - 1).Interface()
}
