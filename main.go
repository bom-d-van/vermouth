package main

import (
	"bytes"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/bom-d-van/goutils/errutils"
)

var (
	gopath      = os.Getenv("GOPATH")
	prevPkgFlag = flag.String("prev", "", "")
	newPkgFlag  = flag.String("new", "", "")
	outputFlag  = flag.String("output", "", "")
)

type (
	Changes struct {
		ModifiedStructs    []*ModifiedStruct
		ModifiedInterfaces []*ModifiedInterface
		RemovedStructs     []*Struct
		RemovedInterfaces  []*Interface
		NewStructs         []*Struct
		NewInterfaces      []*Interface
	}

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

var tmpl = template.Must(template.New("vermouth").Parse(`{{define "changes"}}Structs:

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

Deprecated:

Modified:

New:

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

func (c *Changes) GenDoc() (doc string) {
	if c.isBackwardCompatible() {
		doc += "The New API is backward compatible.\n"
	} else {
		doc += "The New API is NOT backward compatible.\n"
	}
	// doc += "Modified Structs:\n"
	// for _, s := range c.ModifiedStructs {
	// 	doc += s.Name + "\n"
	// 	for _, f := range s.ModifiedFields {
	// 		doc += fmt.Sprintf("%s: %s -> %s", f.Name, f.OriginType, f.CurrentType)
	// 	}
	// }

	w := bytes.NewBufferString("")
	tmpl.ExecuteTemplate(w, "changes", c)
	doc += w.String()

	doc = strings.Replace(doc, "\n\n", "\n", -1)
	println(doc)

	return
}

func (c *Changes) isBackwardCompatible() bool {
	fieldChanged := false
	for _, s := range c.ModifiedStructs {
		fieldChanged = len(s.ModifiedFields) != 0 || len(s.RemovedFields) != 0
		if fieldChanged {
			break
		}
	}
	signatureChanged := false
	for _, f := range c.ModifiedInterfaces {
		_ = f
		// signatureChanged = len(f.ModifiedInputs) != 0 || len(f.ModifiedInputs) != 0
		// if signatureChanged {
		// 	break
		// }
	}
	return len(c.RemovedStructs) == 0 && len(c.RemovedInterfaces) == 0 && !fieldChanged && !signatureChanged
}

func main() {
	flag.Parse()
	// log.Println("parsing packages")
	// changes, err := parse(*prevPkgFlag, *newPkgFlag)
	// if err != nil {
	// 	panic(err)
	// }
}

func parse(prevPkgVal, newPkgVal string) (changes *Changes, err error) {
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
	sf := &structFinder{structs: w.structs}
	ast.Walk(sf, newPkg)
	w.changes.NewStructs = sf.newStructs

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
			// ms := w.compareStructs(t, newSpec.Type.(*ast.StructType))
			msx := w.compareFieldList(t.Fields, newSpec.Type.(*ast.StructType).Fields, true)
			if msx == nil {
				return w
			}

			ms := msx.(*ModifiedStruct)
			ms.Name = name
			w.changes.ModifiedStructs = append(w.changes.ModifiedStructs, ms)
		case *ast.InterfaceType:
			// name := spec.Name.Name
			// log.Println("Parsing", name)
			// w.interfaces += name + ","
			// newSpec := w.finder.byName(name)
			// if newSpec == nil {
			// 	w.changes.RemovedMethods = append(w.changes.RemovedMethods, newMethod(name, t))
			// }
		}
	}

	return w
}

func newMethod(name string, node *ast.InterfaceType) (m *Method) {
	m = new(Method)
	m.Name = name
	return
}

func (w *walker) compareFieldList(ofl, nfl *ast.FieldList, isStruct bool) interface{} {
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
				log.Println("Found a removed field:", ident.Name)
				isChanged = true
				if isStruct {
					ms.RemovedFields = append(ms.RemovedFields, newField(ident.Name, f))
				} else {

				}

				continue
			}

			if !w.isSameType(f.Type, nf.Type) {
				log.Println("Found a type-changed field:", ident.Name)
				isChanged = true
				if isStruct {
					ms.ModifiedFields = append(ms.ModifiedFields, newModifiedField(ident.Name, f.Type, nf.Type))
				} else {

				}
			}
		}
	}

	log.Println("Finding new fields")
	for _, f := range nfl.List {
		for _, ident := range f.Names {
			if !strings.Contains(fields, ident.Name+",") {
				log.Println("Found a new field:", ident.Name)
				isChanged = true
				if isStruct {
					ms.NewFields = append(ms.NewFields, newField(ident.Name, f))
				} else {

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
	return t.(*ast.Ident).Name == nt.(*ast.Ident).Name
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

type structFinder struct {
	structs    string
	newStructs []*Struct
}

func (s *structFinder) Visit(node ast.Node) ast.Visitor {
	switch ts := node.(type) {
	case *ast.TypeSpec:
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return s
		}
		if strings.Contains(s.structs, ts.Name.Name) {
			return s
		}
		log.Println("Found a new struct:", ts.Name.Name)
		s.newStructs = append(s.newStructs, newStruct(ts.Name.Name, st))
	}

	return s
}

func filterStub(os.FileInfo) bool { return true }

func pkgPath(pkg string) string {
	return gopath + "/src/" + pkg
}
