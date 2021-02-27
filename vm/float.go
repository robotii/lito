package vm

import (
	"math"
	"strings"

	"strconv"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// FloatObject represents a real number
type FloatObject float64

var floatClass *RClass

var floatClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn:   NoSuchMethod("new"),
	},
}

var floatInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "+",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightNumeric, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			return FloatObject(float64(receiver.(FloatObject)) + rightNumeric.floatValue())
		},
	},
	{
		Name: "%",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightNumeric, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			if rightNumeric.floatValue() == 0 {
				return t.vm.InitErrorObject(t, errors.ZeroDivisionError, errors.DividedByZero)
			}
			return FloatObject(math.Mod(float64(receiver.(FloatObject)), rightNumeric.floatValue()))
		},
	},
	{
		Name: "-",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightNumeric, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			return FloatObject(float64(receiver.(FloatObject)) - rightNumeric.floatValue())
		},
	},
	{
		Name: "*",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightNumeric, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			return FloatObject(float64(receiver.(FloatObject)) * rightNumeric.floatValue())
		},
	},
	{
		Name: "**",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightNumeric, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			return FloatObject(math.Pow(float64(receiver.(FloatObject)), rightNumeric.floatValue()))
		},
	},
	{
		Name: "/",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightNumeric, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}

			rightValue := rightNumeric.floatValue()
			if rightValue == 0 {
				return t.vm.InitErrorObject(t, errors.ZeroDivisionError, errors.DividedByZero)
			}
			return FloatObject(float64(receiver.(FloatObject)) / rightValue)
		},
	},
	{
		Name: ">",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightObj, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}

			return BooleanObject(float64(receiver.(FloatObject)) > rightObj.floatValue())
		},
	},
	{
		Name: ">=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightObj, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError,
					errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			return BooleanObject(float64(receiver.(FloatObject)) >= rightObj.floatValue())
		},
	},
	{
		Name: "<",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightObj, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError,
					errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			return BooleanObject(float64(receiver.(FloatObject)) < rightObj.floatValue())
		},
	},
	{
		Name: "<=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightObj, ok := args[0].(Numeric)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError,
					errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			return BooleanObject(float64(receiver.(FloatObject)) <= rightObj.floatValue())
		},
	},
	{
		Name: "int",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			r := float64(receiver.(FloatObject))
			return IntegerObject(int(r))
		},
	},
	{
		Name: "json",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			r := receiver.(FloatObject)
			return StringObject(r.ToJSON(t))
		},
	},
	{
		Name: "inf?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return BooleanObject(math.IsInf(float64(receiver.(FloatObject)), 0))
		},
		Primitive: true,
	},
	{
		Name: "nan?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			return BooleanObject(math.IsNaN(float64(receiver.(FloatObject))))
		},
		Primitive: true,
	},
}

func initFloatClass(vm *VM) *RClass {
	floatClass = vm.InitClass(classes.FloatClass).
		ClassMethods(floatClassMethods).
		InstanceMethods(floatInstanceMethods).
		SetConstant("NaN", FloatObject(math.NaN())).
		SetConstant("Inf", FloatObject(math.Inf(0)))
	return floatClass
}

// Value returns the object
func (f FloatObject) Value() interface{} {
	return float64(f)
}

// Numeric interface
func (f FloatObject) floatValue() float64 {
	return float64(f)
}

// EqualTo apply an equality test, returning true if the objects are considered equal,
// and false otherwise.
func (f FloatObject) EqualTo(rightObject Object) bool {
	rightNumeric, ok := rightObject.(Numeric)
	return ok && float64(f) == rightNumeric.floatValue()
}

func (f FloatObject) lessThan(arg Object) bool {
	rightNumeric, ok := arg.(Numeric)
	return ok && f.floatValue() < rightNumeric.floatValue()
}

// ToString returns the object's value as the string format
func (f FloatObject) ToString(t *Thread) string {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)

	if strings.Contains(s, ".") || s == "Inf" || s == "NaN" || s == "-Inf" {
		return s
	}
	// Add ".0" to represent a float number
	return s + ".0"
}

// Inspect delegates to ToString
func (f FloatObject) Inspect(t *Thread) string {
	return f.ToString(t)
}

// ToJSON just delegates to ToString
// Converts NaN and Inf to null, as these are not supported in the JSON spec
func (f FloatObject) ToJSON(t *Thread) string {
	s := f.ToString(t)
	if s == "Inf" || s == "NaN" || s == "-Inf" {
		return "null"
	}
	return s
}

// equal checks if the Float values are equal
func (f FloatObject) equal(e FloatObject) bool {
	return float64(f) == float64(e)
}

// Class ...
func (f FloatObject) Class() *RClass {
	return floatClass
}

// GetVariable ...
func (f FloatObject) GetVariable(string) (Object, bool) {
	return nil, false
}

// SetVariable ...
func (f FloatObject) SetVariable(string, Object) Object {
	return f
}

// FindLookup ...
func (f FloatObject) FindLookup(searchAncestor bool) (method Object) {
	method, _ = f.Class().Methods[lookupMethod]

	if method == nil && searchAncestor {
		method = f.FindMethod(lookupMethod, false)
	}
	return
}

// FindMethod ...
func (f FloatObject) FindMethod(methodName string, super bool) (method Object) {
	if super {
		return f.Class().superClass.lookupMethod(methodName)
	}
	return f.Class().lookupMethod(methodName)
}

// Variables ...
func (f FloatObject) Variables() Environment {
	return nil
}

// SetVariables ...
func (f FloatObject) SetVariables(Environment) {
}

// IsTruthy ...
func (f FloatObject) IsTruthy() bool {
	return float64(f) != 0
}
