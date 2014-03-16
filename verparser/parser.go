package verparser

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/bom-d-van/goutils/errutils"
)

var (
	gopath = os.Getenv("GOPATH")
)

type Changes struct {
	ModifiedStructs    []*ModifiedStruct
	ModifiedInterfaces []*ModifiedInterface
	RemovedStructs     []*Struct
	RemovedInterfaces  []*Interface
	NewStructs         []*Struct
	NewInterfaces      []*Interface
}

func Parse(prevPkgVal, newPkgVal string) (changes *Changes, err error) {
	log.Println("Parsing packages")
	prevPkgs, err := parser.ParseDir(token.NewFileSet(), pkgPath(prevPkgVal), filterStub, parser.ParseComments|parser.AllErrors)
	if err != nil {
		err = errutils.Wrap(err)
		return
	}
	newPkgs, err := parser.ParseDir(token.NewFileSet(), pkgPath(newPkgVal), filterStub, parser.ParseComments|parser.AllErrors)
	if err != nil {
		err = errutils.Wrap(err)
		return
	}
	var prevPkg, newPkg *ast.Package
	for _, pkg := range prevPkgs {
		prevPkg = pkg
	}
	for _, pkg := range newPkgs {
		newPkg = pkg
	}

	log.Println("Walking through old api")
	changes = new(Changes)
	finder := &typeFinder{pkg: newPkg}
	w := &walker{pkg: prevPkg, finder: finder, changes: changes}
	ast.Walk(w, w.pkg)

	log.Println("Finding new structs")
	nf := &newsFinder{structs: w.structs, interfaces: w.interfaces}
	ast.Walk(nf, newPkg)
	w.changes.NewStructs = nf.newStructs
	w.changes.NewInterfaces = nf.newInterfaces

	return
}

func (c *Changes) GenDoc() (doc string) {
	if c.isBackwardCompatible() {
		doc += "The New API is backward compatible.\n"
	} else {
		doc += "The New API is NOT backward compatible.\n"
	}

	w := bytes.NewBufferString("")
	tmpl.ExecuteTemplate(w, "changes", c)
	doc += w.String()

	doc = regexp.MustCompile(`[\t ]+\n`).ReplaceAllString(doc, "\n")
	doc = regexp.MustCompile(`\n{2,}`).ReplaceAllString(doc, "\n")

	return
}

func (c *Changes) isBackwardCompatible() bool {
	for _, s := range c.ModifiedStructs {
		if len(s.ModifiedFields) != 0 || len(s.RemovedFields) != 0 {
			return false
		}
	}
	for _, mi := range c.ModifiedInterfaces {
		if len(mi.ModifiedMethods) != 0 || len(mi.RemovedMethods) != 0 {
			return false
		}
	}

	return len(c.RemovedStructs) == 0 && len(c.RemovedInterfaces) == 0
}

type (
	Field struct {
		Name string
		Type string
	}

	ModifiedField struct {
		Name        string
		OriginType  string
		CurrentType string
	}

	ModifiedStruct struct {
		Name           string
		ModifiedFields []*ModifiedField
		RemovedFields  []*Field
		NewFields      []*Field
	}

	Struct struct {
		Name   string
		Fields []*Field
	}

	ModifiedInterface struct {
		Name            string
		ModifiedMethods []*ModifiedMethod
		RemovedMethods  []*Method
		NewMethods      []*Method
	}

	ModifiedMethod struct {
		OriginalMethod *Method
		CurrentMethod  *Method
	}

	Interface struct {
		Name    string
		Methods []*Method
	}

	Method struct {
		Name    string
		Params  []*Field
		Results []*Field
	}
)

var tmpl = template.Must(template.New("vermouth").Funcs(map[string]interface{}{
	"sigcomma": func(sigs []*Field, index int) string {
		if index+1 == len(sigs) {
			return ""
		}

		return ", "
	},
}).Parse(`{{define "changes"}}Structs:

{{if .RemovedStructs}}Deprecated:{{range .RemovedStructs}} {{.Name}}{{end}}{{end}}
{{if .ModifiedStructs}}
Modified:
{{range .ModifiedStructs}}
{{.Name}}:

{{if .RemovedFields}}
	Deprecated Fields:
	{{range .RemovedFields}}
		{{.Name}} {{.Type}}
	{{end}}
{{end}}

{{if .ModifiedFields}}
	Modified Types:
	{{range .ModifiedFields}}
		{{.Name}}: {{.OriginType}} -> {{.CurrentType}}
	{{end}}
{{end}}

{{if .NewFields}}
	New Fields:
	{{range .NewFields}}
		{{.Name}} {{.Type}}
	{{end}}
{{end}}
{{end}}
{{end}}
{{if .NewStructs}}New:{{range .NewStructs}} {{.Name}}{{end}}{{end}}

========

Interfaces:

{{if .RemovedInterfaces}}Deprecated:{{range .RemovedInterfaces}} {{.Name}}{{end}}{{end}}
{{if .ModifiedInterfaces}}
Modified:
{{range .ModifiedInterfaces}}
{{.Name}}:

{{if .RemovedMethods}}
	Deprecated Methods:
	{{range .RemovedMethods}}
		{{.Name}}({{$p := .Params}}{{range $i, $f := $p}}{{.Name}} {{.Type}}{{sigcomma $p $i}}{{end}}) ({{$p := .Results}}{{range $i, $f := $p}}{{.Name}} {{.Type}}{{sigcomma $p $i}}{{end}})
	{{end}}
{{end}}

{{if .ModifiedMethods}}
	Modified Methods:
	{{range .ModifiedMethods}}
		{{with .OriginalMethod}}{{.Name}}({{$p := .Params}}{{range $i, $f := $p}}{{.Name}} {{.Type}}{{sigcomma $p $i}}{{end}}) ({{$p := .Results}}{{range $i, $f := $p}}{{.Name}} {{.Type}}{{sigcomma $p $i}}{{end}}){{end}}
		->
		{{with .CurrentMethod}}{{.Name}}({{$p := .Params}}{{range $i, $f := $p}}{{.Name}} {{.Type}}{{sigcomma $p $i}}{{end}}) ({{$p := .Results}}{{range $i, $f := $p}}{{.Name}} {{.Type}}{{sigcomma $p $i}}{{end}}){{end}}
	{{end}}
{{end}}

{{if .NewMethods}}
	New Methods:
	{{range .NewMethods}}
		{{.Name}}({{$p := .Params}}{{range $i, $f := $p}}{{.Name}} {{.Type}}{{sigcomma $p $i}}{{end}}) ({{$p := .Results}}{{range $i, $f := $p}}{{.Name}} {{.Type}}{{sigcomma $p $i}}{{end}})
	{{end}}
{{end}}

{{end}}
{{end}}
{{if .NewInterfaces}}New:{{range .NewInterfaces}} {{.Name}}{{end}}{{end}}

{{end}}`))

func newStruct(name string, node *ast.StructType) (s *Struct) {
	s = new(Struct)
	s.Name = name
	// for _, field := range node.Fields.List {
	// }
	return
}

func newField(name string, node *ast.Field) (f *Field) {
	f = new(Field)
	f.Name = name
	f.Type = node.Type.(*ast.Ident).Name

	return
}

func newModifiedField(name string, t, nt ast.Node) (mf *ModifiedField) {
	mf = new(ModifiedField)
	mf.Name = name
	mf.OriginType = t.(*ast.Ident).Name
	mf.CurrentType = nt.(*ast.Ident).Name
	return
}

func newModifiedMethod(name string, mx, nmx ast.Node) (mm *ModifiedMethod) {
	mm = new(ModifiedMethod)
	m, nm := mx.(*ast.FuncType), nmx.(*ast.FuncType)
	mm.OriginalMethod = newMethod(name, m)
	mm.CurrentMethod = newMethod(name, nm)

	return
}

func newInterfaces(name string, node *ast.InterfaceType) (i *Interface) {
	i = new(Interface)
	i.Name = name
	for _, m := range node.Methods.List {
		i.Methods = append(i.Methods, newMethod(m.Names[0].Name, m.Type.(*ast.FuncType)))
	}

	return
}

func newMethod(name string, node *ast.FuncType) (m *Method) {
	m = new(Method)
	m.Name = name
	for _, p := range node.Params.List {
		name := ""
		for _, n := range p.Names {
			if name != "" {
				name += ", "
			}
			name += n.Name
		}
		m.Params = append(m.Params, newField(name, p))
	}
	for _, p := range node.Results.List {
		name := ""
		for _, n := range p.Names {
			if name != "" {
				name += ", "
			}
			name += n.Name
		}
		m.Results = append(m.Results, newField(name, p))
	}

	return
}

type walker struct {
	pkg        *ast.Package // to remove
	finder     *typeFinder
	changes    *Changes
	structs    string
	interfaces string
}

func (w *walker) Visit(nodex ast.Node) ast.Visitor {
	switch spec := nodex.(type) {
	case *ast.TypeSpec:
		switch t := spec.Type.(type) {
		case *ast.StructType:
			name := spec.Name.Name
			log.Println("Parsing", name)
			w.structs += name + ","
			newSpec := w.finder.byName(name)
			if newSpec == nil {
				w.changes.RemovedStructs = append(w.changes.RemovedStructs, newStruct(name, t))

				return w
			}

			log.Println("Comparing", name)
			msx := w.compareFieldList(t, newSpec.Type)
			if msx == nil {
				return w
			}

			ms := msx.(*ModifiedStruct)
			ms.Name = name
			w.changes.ModifiedStructs = append(w.changes.ModifiedStructs, ms)
		case *ast.InterfaceType:
			name := spec.Name.Name
			log.Println("Parsing", name)
			w.interfaces += name + ","
			newSpec := w.finder.byName(name)
			if newSpec == nil {
				w.changes.RemovedInterfaces = append(w.changes.RemovedInterfaces, newInterfaces(name, t))

				return w
			}

			log.Println("Comparing", name)
			mix := w.compareFieldList(t, newSpec.Type)
			if mix == nil {
				return w
			}

			ms := mix.(*ModifiedInterface)
			ms.Name = name
			w.changes.ModifiedInterfaces = append(w.changes.ModifiedInterfaces, ms)
		}
	}

	return w
}

func (w *walker) compareFieldList(onx, nex ast.Node) interface{} {
	var ofl, nfl *ast.FieldList
	var isStruct bool
	switch node := onx.(type) {
	case *ast.StructType:
		isStruct = true
		ofl, nfl = node.Fields, nex.(*ast.StructType).Fields
	case *ast.InterfaceType:
		isStruct = false
		ofl, nfl = node.Methods, nex.(*ast.InterfaceType).Methods
	default:
		return nil
	}

	ms := new(ModifiedStruct)
	mi := new(ModifiedInterface)
	isChanged := false
	fields := ""
	log.Println("Comparing fields")
	for _, f := range ofl.List {
		for _, ident := range f.Names {
			nf := w.findField(nfl, ident.Name)
			fields += ident.Name + ","
			if nf == nil {
				log.Println("Found removed:", ident.Name)
				isChanged = true
				if isStruct {
					ms.RemovedFields = append(ms.RemovedFields, newField(ident.Name, f))
				} else {
					mi.RemovedMethods = append(mi.RemovedMethods, newMethod(ident.Name, f.Type.(*ast.FuncType)))
				}

				continue
			}

			if !w.isSameType(f.Type, nf.Type) {
				log.Println("Found type-changed:", ident.Name)
				isChanged = true
				if isStruct {
					ms.ModifiedFields = append(ms.ModifiedFields, newModifiedField(ident.Name, f.Type, nf.Type))
				} else {
					mi.ModifiedMethods = append(mi.ModifiedMethods, newModifiedMethod(ident.Name, f.Type, nf.Type))
				}
			}
		}
	}

	log.Println("Finding new")
	for _, f := range nfl.List {
		for _, ident := range f.Names {
			if !strings.Contains(fields, ident.Name+",") {
				log.Println("Found :", ident.Name)
				isChanged = true
				if isStruct {
					ms.NewFields = append(ms.NewFields, newField(ident.Name, f))
				} else {
					mi.NewMethods = append(mi.NewMethods, newMethod(ident.Name, f.Type.(*ast.FuncType)))
				}
			}
		}
	}

	if !isChanged {
		return nil
	}

	if isStruct {
		return ms
	} else {
		return mi
	}

	return nil
}

func (w *walker) findField(fl *ast.FieldList, name string) *ast.Field {
	for _, f := range fl.List {
		for _, ident := range f.Names {
			if name == ident.Name {
				return f
			}
		}
	}

	return nil
}

func (w *walker) isSameType(t, nt ast.Node) bool {
	switch tt := t.(type) {
	case *ast.Ident:
		return tt.Name == nt.(*ast.Ident).Name
	case *ast.FuncType:
		ntt := nt.(*ast.FuncType)
		if len(tt.Params.List) != len(ntt.Params.List) || len(tt.Results.List) != len(ntt.Results.List) {
			return false
		}
		for i, p := range tt.Params.List {
			if len(ntt.Params.List[i].Names) != len(p.Names) {
				return false
			}
			for j, name := range p.Names {
				if ntt.Params.List[i].Names[j].Name != name.Name {
					return false
				}
			}
			if !w.isSameType(p.Type, ntt.Params.List[i].Type) {
				return false
			}
		}
		for i, p := range tt.Results.List {
			if len(ntt.Results.List[i].Names) != len(p.Names) {
				return false
			}
			for j, name := range p.Names {
				if ntt.Results.List[i].Names[j].Name != name.Name {
					return false
				}
			}
			if !w.isSameType(p.Type, ntt.Results.List[i].Type) {
				return false
			}
		}

		return true
	}

	return false
}

type typeFinder struct {
	pkg    *ast.Package // to remove
	name   string
	target *ast.TypeSpec
}

func (f *typeFinder) byName(name string) *ast.TypeSpec {
	f.name = name
	f.target = nil
	ast.Walk(f, f.pkg)

	return f.target
}

func (f *typeFinder) Visit(nodex ast.Node) ast.Visitor {
	switch node := nodex.(type) {
	case *ast.TypeSpec:
		if node.Name.Name == f.name {
			f.target = node
			return nil
		}
	}
	return f
}

type newsFinder struct {
	structs       string
	interfaces    string
	newStructs    []*Struct
	newInterfaces []*Interface
}

func (s *newsFinder) Visit(node ast.Node) ast.Visitor {
	ts, ok := node.(*ast.TypeSpec)
	if !ok {
		return s
	}

	switch st := ts.Type.(type) {
	case *ast.StructType:
		if strings.Contains(s.structs, ts.Name.Name+",") {
			return s
		}
		log.Println("Found a new struct:", ts.Name.Name)
		s.newStructs = append(s.newStructs, newStruct(ts.Name.Name, st))
	case *ast.InterfaceType:
		if strings.Contains(s.interfaces, ts.Name.Name+",") {
			return s
		}
		log.Println("Found a new interface:", ts.Name.Name)
		s.newInterfaces = append(s.newInterfaces, newInterfaces(ts.Name.Name, st))
	}

	return s
}

func filterStub(os.FileInfo) bool { return true }

func pkgPath(pkg string) string {
	return gopath + "/src/" + pkg
}
