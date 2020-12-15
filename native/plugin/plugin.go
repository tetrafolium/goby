package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"

	"github.com/st0012/metago"
	"github.com/tetrafolium/goby/vm"
	"github.com/tetrafolium/goby/vm/classes"
	"github.com/tetrafolium/goby/vm/errors"
)

type (
	// BaseObj is an imported object from vm
	BaseObj = vm.BaseObj
	// GoObject is an imported object from vm
	GoObject = vm.GoObject
	// ArrayObject is an imported object from vm
	ArrayObject = vm.ArrayObject
	// StringObject is an imported object from vm
	StringObject = vm.StringObject
	// HashObject is an imported object from vm
	HashObject = vm.HashObject
	// VM is an imported object from vm
	VM = vm.VM
	// Thread is an imported object from vm
	Thread = vm.Thread
	// Method is an imported object from vm
	Method = vm.Method
	// Object is an imported object from vm
	Object = vm.Object
)

var (
	// NULL is an imported constant from vm
	NULL = vm.NULL
)

func init() {
	vm.RegisterExternalClass("plugin", vm.NewExternalClassLoader("Plugin", "plugin.gb",
		// class methods
		map[string]Method{
			"new": newPlugin,
			"use": use,
		},
		// instance methods
		map[string]Method{
			"compile": compile,
			"go_func": goFunc,
		},
	))
}

// PluginObject is a special type that contains a Go's plugin
type PluginObject struct {
	*BaseObj
	fn     string
	plugin *plugin.Plugin
}

func newPlugin(receiver Object, sourceLine int, t *Thread, args []Object) Object {
	if len(args) != 1 {
		return t.VM().InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 1, len(args))
	}

	name, ok := args[0].(*StringObject)

	if !ok {
		return t.VM().InitErrorObject(errors.TypeError, sourceLine, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
	}

	return &PluginObject{fn: name.Value().(string), BaseObj: vm.NewBaseObject(t.VM().TopLevelClass(classes.PluginClass))}
}

func use(receiver Object, sourceLine int, t *Thread, args []Object) Object {
	pkgPath := args[0].(*StringObject).Value().(string)
	_, pkgName := filepath.Split(pkgPath)
	pkgName = strings.Split(pkgName, ".")[0]
	soName := filepath.Join("./", pkgName+".so")

	p, err := compileAndOpenPlugin(soName, pkgPath)

	if err != nil {
		return t.VM().InitErrorObject(errors.InternalError, sourceLine, err.Error())
	}

	return &PluginObject{fn: pkgName, plugin: p, BaseObj: vm.NewBaseObject(t.VM().TopLevelClass(classes.PluginClass))}
}
func compile(receiver Object, sourceLine int, t *Thread, args []Object) Object {
	r := receiver.(*PluginObject)
	context, ok := receiver.InstanceVariableGet("@context")

	if !ok {
		return NULL
	}

	// Create plugins directory
	pluginDir := "./plugins"

	ok, err := fileExists(pluginDir)

	if err != nil {
		return t.VM().InitErrorObject(errors.InternalError, sourceLine, err.Error())
	}

	if !ok {
		os.Mkdir(pluginDir, 0777)
	}

	// generate plugin content from context
	pc := setPluginContext(context)
	pluginContent := compilePluginTemplate(pc.pkgs, pc.funcs)

	// create plugin file
	fn := fmt.Sprintf("%s/%s", pluginDir, r.fn)

	file, err := os.OpenFile(fn+".go", os.O_RDWR|os.O_CREATE, 0755)

	if err != nil {
		return t.VM().InitErrorObject(errors.InternalError, sourceLine, "Error when creating plugin: %s", err.Error())
	}

	file.WriteString(pluginContent)

	soName := fn + ".so"

	p, err := compileAndOpenPlugin(soName, file.Name())

	if err != nil {
		t.VM().InitErrorObject(errors.InternalError, sourceLine, err.Error())
	}

	r.plugin = p

	return r

}

func goFunc(receiver Object, sourceLine int, t *Thread, args []Object) Object {
	s, ok := args[0].(*StringObject)

	if !ok {
		return t.VM().InitErrorObject(errors.TypeError, sourceLine, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
	}

	funcName := s.Value().(string)
	r := receiver.(*PluginObject)
	p := r.plugin
	f, err := p.Lookup(funcName)

	if err != nil {
		return t.VM().InitErrorObject(errors.InternalError, sourceLine, err.Error())
	}

	funcArgs, err := vm.ConvertToGoFuncArgs(args[1:])

	if err != nil {
		t.VM().InitErrorObject(errors.TypeError, sourceLine, err.Error())
	}

	funcValue := reflect.ValueOf(f)

	// Check if f is a pointer to function instead of function object
	if funcValue.Type().Kind() == reflect.Ptr {
		ptr := funcValue
		funcValue = ptr.Elem()
	}

	result := reflect.ValueOf(funcValue.Call(metago.WrapArguments(funcArgs...))).Interface()

	return t.VM().InitObjectFromGoType(metago.UnwrapReflectValues(result))
}

// ToString returns the object's name as the string format
func (p *PluginObject) ToString() string {
	return "<Plugin: " + p.fn + ">"
}

// Inspect delegates to ToString
func (p *PluginObject) Inspect() string {
	return p.ToString()
}

// ToJSON just delegates to ToString
func (p *PluginObject) ToJSON(t *Thread) string {
	return p.ToString()
}

// Value returns plugin object's string format
func (p *PluginObject) Value() interface{} {
	return p.plugin
}

// Other helper functions -----------------------------------------------

func setPluginContext(context Object) *pluginContext {
	pc := &pluginContext{pkgs: []*pkg{}, funcs: []*function{}}

	funcs, _ := context.InstanceVariableGet("@functions")
	pkgs, _ := context.InstanceVariableGet("@packages")

	fs := funcs.(*ArrayObject)
	ps := pkgs.(*ArrayObject)

	for _, f := range fs.Elements {
		fInfos := f.(*HashObject)
		prefix := fInfos.Pairs["prefix"].(*StringObject).Value().(string)
		name := fInfos.Pairs["name"].(*StringObject).Value().(string)

		pc.addFunc(prefix, name)
	}

	for _, p := range ps.Elements {
		pInfos := p.(*HashObject)
		prefix := pInfos.Pairs["prefix"].(*StringObject).Value().(string)
		name := pInfos.Pairs["name"].(*StringObject).Value().(string)

		pc.importPkg(prefix, name)
	}

	return pc
}

// fileExists returns whether the given file or directory exists or not
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func compileAndOpenPlugin(soName, fileName string) (*plugin.Plugin, error) {
	// Open plugin first
	p, err := plugin.Open(soName)

	// If there's any issue open a plugin, assume it's not well compiled
	if err != nil {
		cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", soName, fileName)
		out, err := cmd.CombinedOutput()

		if err != nil {
			return nil, fmt.Errorf("Error: %s from %s", string(out), strings.Join(cmd.Args, " "))
		}

		p, err = plugin.Open(soName)

		if err != nil {
			return nil, fmt.Errorf("Error occurs when open %s package: %s", soName, err.Error())
		}
	}

	return p, nil
}

// Plugin context =======================================================

type pluginContext struct {
	pkgs  []*pkg
	funcs []*function
}

// Polymorphic helper functions -----------------------------------------

func (c *pluginContext) importPkg(prefix, name string) {
	c.pkgs = append(c.pkgs, &pkg{Prefix: prefix, Name: name})
}

func (c *pluginContext) addFunc(prefix, name string) {
	c.funcs = append(c.funcs, &function{Prefix: prefix, Name: name})
}
