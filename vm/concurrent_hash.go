package vm

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/tetrafolium/goby/vm/classes"
	"github.com/tetrafolium/goby/vm/errors"
)

// ConcurrentHashObject is an implementation of thread-safe associative arrays (Hash).
//
// The implementation internally uses Go's `sync.Map` type, with some advantages and disadvantages:
//
// - it is highly performant and predictable for a certain pattern of usage (`concurrent loops with keys that are stable over time, and either few steady-state stores, or stores localized to one goroutine per key.`); performance and predictability in other conditions are unspecified;
// - iterations are non-deterministic; during iterations, keys may not be included;
// - size can't be retrieved;
// - for the reasons above, the Hash APIs implemented are minimal.
//
// For details, see https://golang.org/pkg/sync/#Map.
//
// ```ruby
// require 'concurrent/hash'
// hash = Concurrent::Hash.new({ "a": 1, "b": 2 })
// hash["a"]  # => 1
// ```
//
type ConcurrentHashObject struct {
	*BaseObj
	internalMap *sync.Map
}

// Class methods --------------------------------------------------------
var builtinConcurrentHashClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			aLen := len(args)
			if aLen > 1 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgumentLess, 1, aLen)
			}

			if aLen == 0 {
				return t.vm.initConcurrentHashObject(make(map[string]Object))
			}

			hashArg, ok := args[0].(*HashObject)

			if !ok {
				return t.vm.InitErrorObject(errors.TypeError, sourceLine, errors.WrongArgumentTypeFormat, classes.HashClass, args[0].Class().Name)
			}

			return t.vm.initConcurrentHashObject(hashArg.Pairs)

		},
	},
}

// Instance methods -----------------------------------------------------
var builtinConcurrentHashInstanceMethods = []*BuiltinMethodObject{
	{
		// Retrieves the value (object) that corresponds to the key specified.
		// When a key doesn't exist, `nil` is returned, or the default, if set.
		//
		// ```Ruby
		// h = Concurrent::Hash.new({ a: 1, b: "2" })
		// h['a'] #=> 1
		// h['b'] #=> "2"
		// h['c'] #=> nil
		// ```
		//
		// @return [Object]
		Name: "[]",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 1, len(args))
			}

			err := t.vm.checkArgTypes(args, sourceLine, classes.StringClass)

			if err != nil {
				return err
			}

			h := receiver.(*ConcurrentHashObject)

			value, ok := h.internalMap.Load(args[0].Value().(string))

			if !ok {
				return NULL
			}

			return value.(Object)

		},
	},
	{
		// Associates the value given by `value` with the key given by `key`.
		// Returns the `value`.
		//
		// ```Ruby
		// h = Concurrent::Hash.new{ a: 1, b: "2" })
		// h['a'] = 2          #=> 2
		// h                   #=> { a: 2, b: "2" }
		// ```
		//
		// @return [Object] The value
		Name: "[]=",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			// First arg is index
			// Second arg is assigned value
			if len(args) != 2 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 2, len(args))
			}

			err := t.vm.checkArgTypes(args, sourceLine, classes.StringClass)

			if err != nil {
				return err
			}

			h := receiver.(*ConcurrentHashObject)
			h.internalMap.Store(args[0].Value().(string), args[1])

			return args[1]

		},
	},
	{
		// Remove the key from the hash if key exist.
		//
		// ```Ruby
		// h = Concurrent::Hash.new({ a: 1, b: 2, c: 3 })
		// h.delete("b") # => NULL
		// h             # => { a: 1, c: 3 }
		// ```
		//
		// @return [NULL]
		Name: "delete",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 1, len(args))
			}

			err := t.vm.checkArgTypes(args, sourceLine, classes.StringClass)

			if err != nil {
				return err
			}

			receiver.(*ConcurrentHashObject).internalMap.Delete(args[0].Value().(string))

			return NULL

		},
	},
	{
		// Calls block once for each key in the hash (in sorted key order), passing the
		// key-value pair as parameters.
		// Note that iteration is not deterministic under all circumstances; see
		// https://golang.org/pkg/sync/#Map.
		//
		// ```Ruby
		// h = Concurrent::Hash.new({ b: "2", a: 1 })
		// h.each do |k, v|
		//   puts k.to_s + "->" + v.to_s
		// end
		// # => a->1
		// # => b->2
		// ```
		//
		// @return [Hash] self
		Name: "each",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 0, len(args))
			}

			if blockFrame == nil {
				return t.vm.InitErrorObject(errors.InternalError, sourceLine, errors.CantYieldWithoutBlockFormat)
			}

			hash := receiver.(*ConcurrentHashObject)
			framePopped := false

			iterator := func(key, value interface{}) bool {
				keyObject := t.vm.InitStringObject(key.(string))

				t.builtinMethodYield(blockFrame, keyObject, value.(Object))

				framePopped = true

				return true
			}

			hash.internalMap.Range(iterator)

			if !framePopped {
				t.callFrameStack.pop()
			}

			return hash

		},
	},
	{
		// Returns true if the key exist in the hash.
		//
		// ```Ruby
		// h = Concurrent::Hash.new({ a: 1, b: "2" })
		// h.has_key?("a") # => true
		// h.has_key?("e") # => false
		// ```
		//
		// @return [Boolean]
		Name: "has_key?",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 1, len(args))
			}

			err := t.vm.checkArgTypes(args, sourceLine, classes.StringClass)

			if err != nil {
				return err
			}

			if _, ok := receiver.(*ConcurrentHashObject).internalMap.Load(args[0].Value().(string)); ok {
				return TRUE
			}

			return FALSE

		},
	},
	{
		// Returns json that is corresponding to the hash.
		// Basically just like Hash#to_json in Rails but currently doesn't support options.
		//
		// ```Ruby
		// h = Concurrent::Hash.new({ a: 1, b: 2 })
		// h.to_json #=> {"a":1,"b":2}
		// ```
		//
		// @return [String]
		Name: "to_json",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 0, len(args))
			}

			r := receiver.(*ConcurrentHashObject)
			return t.vm.InitStringObject(r.ToJSON(t))

		},
	},
	{
		// Returns json that is corresponding to the hash.
		// Basically just like Hash#to_json in Rails but currently doesn't support options.
		//
		// ```Ruby
		// h = Concurrent::Hash.new({ a: 1, b: "2"})
		// h.to_s #=> "{ a: 1, b: \"2\" }"
		// ```
		//
		// @return [String]
		Name: "to_s",
		Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 0, len(args))
			}

			h := receiver.(*ConcurrentHashObject)
			return t.vm.InitStringObject(h.ToString())

		},
	},
}

// Internal functions ===================================================

// Functions for initialization -----------------------------------------

func (vm *VM) initConcurrentHashObject(pairs map[string]Object) *ConcurrentHashObject {
	var internalMap sync.Map

	for key, value := range pairs {
		internalMap.Store(key, value)
	}

	concurrent := vm.loadConstant("Concurrent", true)

	return &ConcurrentHashObject{
		BaseObj:     NewBaseObject(concurrent.getClassConstant(classes.HashClass)),
		internalMap: &internalMap,
	}
}

func initConcurrentHashClass(vm *VM) {
	concurrent := vm.loadConstant("Concurrent", true)
	hash := vm.initializeClass(classes.HashClass)

	hash.setBuiltinMethods(builtinConcurrentHashInstanceMethods, false)
	hash.setBuiltinMethods(builtinConcurrentHashClassMethods, true)

	concurrent.setClassConstant(hash)
}

// Polymorphic helper functions -----------------------------------------

// Value returns the object
func (h *ConcurrentHashObject) Value() interface{} {
	return h.internalMap
}

// ToString returns the object's name as the string format
func (h *ConcurrentHashObject) ToString() string {
	var out bytes.Buffer
	var pairs []string

	iterator := func(key, value interface{}) bool {
		pairs = append(pairs, fmt.Sprintf("%s: %s", key, value.(Object).Inspect()))
		return true
	}

	h.internalMap.Range(iterator)

	out.WriteString("{ ")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString(" }")

	return out.String()
}

// Inspect delegates to ToString
func (h *ConcurrentHashObject) Inspect() string {
	return h.ToString()
}

// ToJSON returns the object's name as the JSON string format
func (h *ConcurrentHashObject) ToJSON(t *Thread) string {
	var out bytes.Buffer
	var values []string
	out.WriteString("{")

	iterator := func(key, value interface{}) bool {
		values = append(values, generateJSONFromPair(key.(string), value.(Object), t))

		return true
	}

	h.internalMap.Range(iterator)

	out.WriteString(strings.Join(values, ","))
	out.WriteString("}")
	return out.String()
}
