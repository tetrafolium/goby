package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/fatih/camelcase"

	// makes writing this easier
	"github.com/dave/jennifer/jen"
)

var (
	in       = flag.String("in", "", "folder to create bindings from")
	typeName = flag.String("type", "", "type to generate bindings for")
)

const (
	vmPkg     = "github.com/tetrafolium/goby/vm"
	errorsPkg = "github.com/tetrafolium/goby/vm/errors"
)

func typeFromExpr(e ast.Expr) string {
	var name string
	switch t := e.(type) {
	case *ast.Ident:
		name = t.Name

	case *ast.StarExpr:
		name = fmt.Sprintf("*%s", typeFromExpr(t.X))

	case *ast.SelectorExpr:
		name = fmt.Sprintf("%s.%s", typeFromExpr(t.X), t.Sel.Name)

	}
	return name
}

func typeNameFromExpr(e ast.Expr) string {
	var name string
	switch t := e.(type) {
	case *ast.Ident:
		name = t.Name

	case *ast.StarExpr:
		name = typeFromExpr(t.X)

	case *ast.SelectorExpr:
		name = fmt.Sprintf("%s.%s", typeFromExpr(t.X), t.Sel.Name)

	}
	return name
}

type argPair struct {
	name, kind string
}

func allArgs(f *ast.FieldList) []argPair {
	var args []argPair
	for _, l := range f.List {
		for _, n := range l.Names {
			args = append(args, argPair{
				name: n.Name,
				kind: typeNameFromExpr(l.Type),
			})
		}
	}

	return args
}

// Binding holds context about a struct that represents a goby class.
type Binding struct {
	ClassName       string
	ClassMethods    []*ast.FuncDecl // Any method defined without a pointer receiver is a class method func (Class) myFunc
	InstanceMethods []*ast.FuncDecl // Any method defined with a pointer receiver is an instance method func (c *Class) myFunc

}

func (b *Binding) topCommentBlock() jen.Code {
	return jen.Add(
		jen.Comment("DO NOT EDIT THIS FILE MANUALLY"),
		jen.Line(),
		jen.Commentf("This code has been generated by github.com/tetrafolium/goby/cmd/binder"),
		jen.Line(),
	)

}

func (b *Binding) staticName() string {
	return fmt.Sprintf("static%s", b.ClassName)
}

func (b *Binding) bindingName(f *ast.FuncDecl) string {
	return fmt.Sprintf("binding%s%s", b.ClassName, f.Name.Name)
}

// BindMethods generates code that binds methods of a go structure to a goby class
func (b *Binding) BindMethods(f *jen.File, x *ast.File) {
	f.Add(b.topCommentBlock())
	f.Add(mapping(b, x.Name.Name))
	f.Var().Id(b.staticName()).Op("=").New(jen.Id(b.ClassName))
	for _, c := range b.ClassMethods {
		f.Commentf("%s is a class method binding for %s.%s", b.bindingName(c), b.ClassName, c.Name.Name)
		b.BindClassMethod(f, c)
		f.Line()
	}
	for _, c := range b.InstanceMethods {
		f.Commentf("%s is an instance method binding for *%s.%s", b.bindingName(c), b.ClassName, c.Name.Name)
		b.BindInstanceMethod(f, c)
		f.Line()
	}
}

// BindClassMethod will generate class method bindings.
// This is a global static method associated with the class.
func (b *Binding) BindClassMethod(f *jen.File, d *ast.FuncDecl) {
	r := jen.Id("r").Op(":=").Id(b.staticName()).Line()
	b.body(r, f, d)
}

// BindInstanceMethod will generate instance method bindings.
// This function will be bound to a spesific instantation of a goby class.
func (b *Binding) BindInstanceMethod(f *jen.File, d *ast.FuncDecl) {
	r := jen.List(jen.Id("r"), jen.Id("ok")).Op(":=").Add(jen.Id("receiver")).Assert(jen.Op("*").Id(b.ClassName)).Line()
	r = r.If(jen.Op("!").Id("ok")).Block(
		jen.Panic(
			jen.Qual("fmt", "Sprintf").Call(jen.Lit("Impossible receiver type. Wanted "+b.ClassName+" got %s"), jen.Id("receiver")),
		),
	).Line()
	b.body(r, f, d)
}

func wrongArgNum(want int) jen.Code {
	return jen.Return(jen.Id("t").Dot("VM").Call().Dot("InitErrorObject").Call(
		jen.Qual(errorsPkg, "ArgumentError"),
		jen.Id("line"),
		jen.Qual(errorsPkg, "WrongNumberOfArgumentFormat"),
		jen.Lit(want),
		jen.Id("len").Call(jen.Id("args")),
	))
}

func wrongArgType(name, want string) jen.Code {
	return jen.Return(jen.Id("t").Dot("VM").Call().Dot("InitErrorObject").Call(
		jen.Qual(errorsPkg, "TypeError"),
		jen.Id("line"),
		jen.Qual(errorsPkg, "WrongArgumentTypeFormat"),
		jen.Lit(want),
		jen.Id(name).Dot("Class").Call().Dot("Name"),
	))
}

// body is a helper function for generating the common body of a method
func (b *Binding) body(receiver *jen.Statement, f *jen.File, d *ast.FuncDecl) {
	s := f.Func().Id(b.bindingName(d))
	s = s.Params(
		jen.Id("receiver").Qual(vmPkg, "Object"),
		jen.Id("line").Id("int"),
		jen.Id("t").Op("*").Qual(vmPkg, "Thread"),
		jen.Id("args").Index().Qual(vmPkg, "Object"),
	).Qual(vmPkg, "Object")

	var args []*jen.Statement
	for i, a := range allArgs(d.Type.Params) {
		if i == 0 {
			continue
		}
		i--
		c := jen.List(jen.Id(fmt.Sprintf("arg%d", i)), jen.Id("ok")).Op(":=").Id("args").Index(jen.Lit(i)).Assert(jen.Id(a.kind))
		c = c.Line()
		c = c.If(jen.Op("!").Id("ok")).Block(
			wrongArgType(fmt.Sprintf("args[%d]", i), a.kind),
		).Line()
		args = append(args, c)
	}

	inner := receiver.If(jen.Len(jen.Id("args")).Op("!=").Lit(d.Type.Params.NumFields() - 1)).Block(
		wrongArgNum(d.Type.Params.NumFields() - 1),
	).Line()
	argNames := []jen.Code{
		jen.Id("t"),
	}
	for i, a := range args {
		inner = inner.Add(a).Line()
		argNames = append(argNames, jen.Id(fmt.Sprintf("arg%d", i)))
	}

	inner = inner.Return(jen.Id("r").Dot(d.Name.Name).Call(argNames...))
	s.Block(inner)
}

// mapping generates the "init" portion of the bindings.
// This will call hooks in the vm package to load the class definition at runtime.
func mapping(b *Binding, pkg string) jen.Code {
	fnName := func(s string) string {
		x := camelcase.Split(s)
		return strings.ToLower(strings.Join(x, "_"))
	}

	cm := jen.Dict{}
	for _, d := range b.ClassMethods {
		cm[jen.Lit(fnName(d.Name.Name))] = jen.Id(b.bindingName(d))
	}
	im := jen.Dict{}
	for _, d := range b.InstanceMethods {
		im[jen.Lit(fnName(d.Name.Name))] = jen.Id(b.bindingName(d))
	}
	dm := jen.Qual(vmPkg, "RegisterExternalClass").Call(
		jen.Line().Lit(pkg),
		jen.Qual(vmPkg, "ExternalClass").Call(
			jen.Line().Lit(b.ClassName),
			jen.Line().Lit(pkg+".gb"),
			jen.Line().Map(jen.String()).Qual(vmPkg, "Method").Values(cm),
			jen.Line().Map(jen.String()).Qual(vmPkg, "Method").Values(im),
		),
	)
	l := jen.Func().Id("init").Params().Block(
		dm,
	)
	return l
}

func main() {
	flag.Usage = func() {
		fmt.Println("binder is used for generating class bindings for go structures.")
		flag.PrintDefaults()
	}

	flag.Parse()
	if *in == "" {
		flag.Usage()
		os.Exit(0)
	}

	fs := token.NewFileSet()
	buff, err := ioutil.ReadFile(*in)
	if err != nil {
		log.Fatal(err)
	}

	f, err := parser.ParseFile(fs, *in, string(buff), parser.AllErrors)
	if err != nil {
		log.Fatal(err)
	}

	bindings := make(map[string]*Binding)

	// iterate though every node in the ast looking for function definitions
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Recv != nil {
				res := n.Type.Results
				if res == nil {
					return true
				}

				if len(res.List) == 0 || typeNameFromExpr(res.List[0].Type) != "Object" {
					return true
				}

				// class or instance?
				r := n.Recv.List[0]
				name := typeNameFromExpr(r.Type)

				b, ok := bindings[name]
				if !ok {
					b = new(Binding)
					b.ClassName = name
					bindings[name] = b
				}

				// class
				if r.Names == nil {
					b.ClassMethods = append(b.ClassMethods, n)
				} else {
					b.InstanceMethods = append(b.InstanceMethods, n)
				}
			}
		case *ast.TypeSpec:
			bindings[n.Name.Name] = &Binding{
				ClassName: n.Name.Name,
			}

		}

		return true
	})

	bnd, ok := bindings[*typeName]
	if !ok {
		log.Fatal("Uknown type", *typeName)
	}

	o := jen.NewFile(f.Name.Name)
	bnd.BindMethods(o, f)

	err = o.Save("bindings.go")
	if err != nil {
		log.Fatal(err)
	}
}
