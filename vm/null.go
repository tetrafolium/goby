package vm

import (
	"github.com/tetrafolium/goby/vm/classes"
	"github.com/tetrafolium/goby/vm/errors"
)

// NullObject (`nil`) represents the null value in Goby.
// `nil` is convert into `null` when exported to JSON format.
// - `Null.new` is not supported.
type NullObject struct {
	*BaseObj
}

var (
	// NULL represents Goby's null objects.
	NULL *NullObject
)

// Class methods --------------------------------------------------------
var builtinNullClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			return t.vm.InitNoMethodError(sourceLine, "new", receiver)

		},
	},
}

// Instance methods -----------------------------------------------------
var builtinNullInstanceMethods = []*BuiltinMethodObject{
	{
		// Returns true: the flipped boolean value of nil object.
		//
		// ```ruby
		// a = nil
		// !a
		// # => true
		// ```
		Name: "!",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {

			return TRUE

		},
	},
	{
		Name: "to_i",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 0, len(args))
			}

			return t.vm.InitIntegerObject(0)

		},
	},
	{
		Name: "to_s",

		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 0, len(args))
			}

			n := receiver.(*NullObject)
			return t.vm.InitStringObject(n.ToString())
		},
	},
	{
		Name: "inspect",

		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, "Expect 0 argument. got: %d", len(args))
			}

			n := receiver.(*NullObject)
			return t.vm.InitStringObject(n.Inspect())
		},
	},
	{
		// Returns true: the flipped boolean value of nil object.
		//
		// ```ruby
		// a = nil
		// a != nil
		// # => false
		// ```
		Name: "!=",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, "Expect 1 argument. got: %d", len(args))
			}

			if _, ok := args[0].(*NullObject); !ok {
				return TRUE
			}
			return FALSE

		},
	},
	{
		// Returns true because it is nil.
		//
		// ```ruby
		// a = nil
		// a.nil?
		// # => true
		// ```
		Name: "nil?",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, "Expect 0 argument. got: %d", len(args))
			}
			return TRUE

		},
	},
}

// Internal functions ===================================================

// Functions for initialization -----------------------------------------

func (vm *VM) initNullClass() *RClass {
	nc := vm.initializeClass(classes.NullClass)
	nc.setBuiltinMethods(builtinNullInstanceMethods, false)
	nc.setBuiltinMethods(builtinNullClassMethods, true)
	NULL = &NullObject{BaseObj: NewBaseObject(nc)}
	return nc
}

// Polymorphic helper functions -----------------------------------------

// Value returns the object
func (n *NullObject) Value() interface{} {
	return nil
}

// ToString returns the object's name as the string format
func (n *NullObject) ToString() string {
	return ""
}

// ToJSON just delegates to ToString
func (n *NullObject) ToJSON(t *Thread) string {
	return "null"
}

// Inspect returns string "nil" instead of "" like ToString
func (n *NullObject) Inspect() string {
	return "nil"
}

func (n *NullObject) isTruthy() bool {
	return false
}

func (n *NullObject) equalTo(compared Object) bool {
	return n == compared
}
