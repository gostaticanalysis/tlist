package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"

	"github.com/gostaticanalysis/knife"
)

var tmpl = `{{with index .Types (data "type")}}{{if interface .}}
// Code generated by mockgen; DO NOT EDIT.

package {{(pkg).Name}}

type Mock{{data "type"}} struct {
{{- range $n, $f := methods .}}
	{{$n}}Func {{$f.Signature}}
{{- end}}
}

{{range $n, $f := methods .}}
func (m *Mock{{data "type"}}) {{$n}}({{range $f.Signature.Params}}
	{{- .Name}} {{.Type}},
{{- end}}) ({{range $f.Signature.Results}}
	{{- .Name}} {{.Type}},
{{- end}}) {
	{{if $f.Signature.Results}}return {{end}}m.{{$n}}Func({{range $f.Signature.Params}}
		{{- .Name}},
	{{- end}})
}
{{end}}
{{end}}
{{end}}
`

var (
	flagOut string
)

func init() {
	flag.StringVar(&flagOut, "o", "", "output file path")
	flag.Parse()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (rerr error) {
	if flag.NArg() < 1 {
		return errors.New("type must be specified")
	}

	k, err := knife.New(flag.Args()[1:]...)
	if err != nil {
		return fmt.Errorf("cannot create knife: %w", err)
	}

	opt := &knife.Option{
		ExtraData: map[string]interface{}{
			"type": flag.Arg(0),
		},
	}

	pkgs := k.Packages()
	if len(pkgs) == 0 {
		return errors.New("does not find package")
	}

	pkg := pkgs[0]
	var buf bytes.Buffer
	if err := k.Execute(&buf, pkg, tmpl, opt); err != nil {
		return fmt.Errorf("cannot knife execute: %w", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("cannot format: %w", err)
	}

	if len(bytes.TrimSpace(src)) == 0 {
		return nil
	}

	var w io.Writer = os.Stdout
	if flagOut != "" {
		f, err := os.Create(flagOut)
		if err != nil {
			return fmt.Errorf("cannot create file: %w", err)
		}
		defer func() {
			if err := f.Close(); err != nil && rerr == nil {
				rerr = err
			}
		}()
		w = f
	}
	if _, err := fmt.Fprintln(w, string(src)); err != nil {
		return fmt.Errorf("cannot output source: %w", err)
	}

	return nil
}