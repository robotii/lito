package vm

import (
	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// NilObject (`nil`) represents the nil value in Lito.
// `nil` is converted into `null` when exported to JSON format.
type NilObject struct {
	BaseObj
}

var (
	// NIL represents Lito's nil object. This is a singleton value
	// and can be compared using ==.
	NIL *NilObject
)

var nilClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn:   NoSuchMethod("new"),
	},
}

var nilInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "!",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return TRUE
		},
	},
	{
		Name: "int",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.VM().InitErrorObject(t, errors.ArgumentError,
					errors.WrongNumberOfArgument, 0, len(args))
			}
			return IntegerObject(0)
		},
	},
	{
		Name: "string",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.VM().InitErrorObject(t, errors.ArgumentError,
					errors.WrongNumberOfArgument, 0, len(args))
			}

			n := receiver.(*NilObject)
			return StringObject(n.ToString(t))
		},
	},
	{
		Name: "inspect",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.VM().InitErrorObject(t, errors.ArgumentError,
					errors.WrongNumberOfArgument, 0, len(args))
			}

			n := receiver.(*NilObject)
			return StringObject(n.Inspect(t))
		},
	},
	{
		Name: "!=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.VM().InitErrorObject(t, errors.ArgumentError, "Expect 1 argument. got: %d", len(args))
			}

			if _, ok := args[0].(*NilObject); !ok {
				return TRUE
			}
			return FALSE
		},
	},
	{
		Name: "nil?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.VM().InitErrorObject(t, errors.ArgumentError, "Expect 0 argument. got: %d", len(args))
			}
			return TRUE
		},
	},
}

func initNilClass(vm *VM) *RClass {
	nc := vm.InitClass(classes.NilClass).
		ClassMethods(nilClassMethods).
		InstanceMethods(nilInstanceMethods)
	NIL = &NilObject{BaseObj: BaseObj{class: nc}}
	return nc
}

// Value returns the object
func (n *NilObject) Value() interface{} {
	return nil
}

// ToString returns the object's name as the string format
func (n *NilObject) ToString(t *Thread) string {
	return ""
}

// ToJSON just delegates to ToString
func (n *NilObject) ToJSON(t *Thread) string {
	return "null"
}

// Inspect returns string "nil" instead of "" like ToString
func (n *NilObject) Inspect(t *Thread) string {
	return "nil"
}

// IsTruthy ...
func (n *NilObject) IsTruthy() bool {
	return false
}

// EqualTo returns if the NilObject is equal to another object.
// This will only return true if the other object is NIL
func (n *NilObject) EqualTo(compared Object) bool {
	return n == compared
}
