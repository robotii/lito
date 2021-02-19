package vm

import (
	"fmt"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// BooleanObject represents boolean object in Lito.
// `Boolean` class is just a dummy to hold logical `true` and `false`
// representation and no other active usage.
type BooleanObject bool

var boolClass *RClass

const (
	// TRUE is shared boolean object that represents true
	TRUE = BooleanObject(true)
	// FALSE is shared boolean object that represents false
	FALSE = BooleanObject(false)
)

var booleanClassMethods = []*BuiltinMethodObject{
	{
		Name:      "new",
		Fn:        NoSuchMethod("new"),
		Primitive: true,
	},
}

var booleanInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "int",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			// Check receiever is boolean when converting
			if b, ok := receiver.(BooleanObject); ok && bool(b) {
				return IntegerObject(1)
			}
			return IntegerObject(0)
		},
		Primitive: true,
	},
	{
		Name: "ifTrue",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return yieldIf(receiver, t, true)
		},
	},
	{
		Name: "ifFalse",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return yieldIf(receiver, t, false)
		},
	},
}

func yieldIf(receiver Object, t *Thread, expected bool) Object {
	blockFrame := t.GetBlock()
	if blockFrame == nil {
		return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
	}

	if receiver.IsTruthy() == expected {
		if blockFrame.IsEmpty() {
			return NIL
		}
		return t.Yield(blockFrame)
	}
	return NIL
}

func initBoolClass(vm *VM) *RClass {
	boolClass = vm.InitClass(classes.BooleanClass).
		ClassMethods(booleanClassMethods).
		InstanceMethods(booleanInstanceMethods)
	return boolClass
}

// Value returns the object
func (b BooleanObject) Value() interface{} {
	return bool(b)
}

// ToString returns the object's name as the string format
func (b BooleanObject) ToString(t *Thread) string {
	return fmt.Sprintf("%t", b)
}

// Inspect delegates to ToString
func (b BooleanObject) Inspect(t *Thread) string {
	return b.ToString(t)
}

// ToJSON just delegates to `ToString`
func (b BooleanObject) ToJSON(t *Thread) string {
	return b.ToString(t)
}

// IsTruthy returns the boolean value of the object
func (b BooleanObject) IsTruthy() bool {
	return bool(b)
}

// EqualTo returns if the BooleanObject is equal to another object
func (b BooleanObject) EqualTo(with Object) bool {
	boolean, ok := with.(BooleanObject)
	return ok && bool(b) == bool(boolean)
}

// equal returns true if the Boolean values between receiver and parameter are equal
func (b BooleanObject) equal(e BooleanObject) bool {
	return b == e
}

// Class ...
func (b BooleanObject) Class() *RClass {
	return boolClass
}

// GetVariable ...
func (b BooleanObject) GetVariable(string) (Object, bool) {
	return nil, false
}

// SetVariable ...
func (b BooleanObject) SetVariable(n string, o Object) Object {
	return o
}

// FindLookup ...
func (b BooleanObject) FindLookup(searchAncestor bool) (method Object) {
	method, _ = b.Class().Methods[lookupMethod]

	if method == nil && searchAncestor {
		method = b.FindMethod(lookupMethod, false)
	}

	return
}

// FindMethod ...
func (b BooleanObject) FindMethod(methodName string, super bool) Object {
	if super {
		return b.Class().superClass.lookupMethod(methodName)
	}
	return b.Class().lookupMethod(methodName)
}

// Variables ...
func (b BooleanObject) Variables() Environment {
	return nil
}

// SetVariables ...
func (b BooleanObject) SetVariables(Environment) {
}
