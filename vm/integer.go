package vm

import (
	"math"
	"strconv"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// IntegerObject represents number objects which can bring into mathematical calculations.
type IntegerObject int

var intClass *RClass

var integerClassMethods = []*BuiltinMethodObject{
	{
		Name:      "new",
		Fn:        NoSuchMethod("new"),
		Primitive: true,
	},
}

var integerInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "+",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				return IntegerObject(int(i) + int(rightObject))
			case FloatObject:
				return FloatObject(i.floatValue() + float64(rightObject))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: "%",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				if int(rightObject) == 0 {
					return t.vm.InitErrorObject(t, errors.ZeroDivisionError, errors.DividedByZero)
				}
				return IntegerObject(int(i) % int(rightObject))
			case FloatObject:
				if float64(rightObject) == 0 {
					return t.vm.InitErrorObject(t, errors.ZeroDivisionError, errors.DividedByZero)
				}
				return FloatObject(math.Mod(i.floatValue(), float64(rightObject)))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: "-",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				return IntegerObject(int(i) - int(rightObject))
			case FloatObject:
				return FloatObject(i.floatValue() - float64(rightObject))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: "*",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				return IntegerObject(int(i) * int(rightObject))
			case FloatObject:
				return FloatObject(i.floatValue() * float64(rightObject))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: "**",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				return IntegerObject(int(math.Pow(float64(i), float64(rightObject))))
			case FloatObject:
				return FloatObject(math.Pow(i.floatValue(), float64(rightObject)))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: "/",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				if int(rightObject) == 0 {
					return t.vm.InitErrorObject(t, errors.ZeroDivisionError, errors.DividedByZero)
				}
				return IntegerObject(int(i) / int(rightObject))
			case FloatObject:
				if float64(rightObject) == 0 {
					return t.vm.InitErrorObject(t, errors.ZeroDivisionError, errors.DividedByZero)
				}
				return FloatObject(i.floatValue() / float64(rightObject))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: ">",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				return BooleanObject(int(i) > int(rightObject))
			case FloatObject:
				return BooleanObject(i.floatValue() > float64(rightObject))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: ">=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				return BooleanObject(int(i) >= int(rightObject))
			case FloatObject:
				return BooleanObject(i.floatValue() >= float64(rightObject))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: "<",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch arg := args[0].(type) {
			case IntegerObject:
				return BooleanObject(int(i) < int(arg))
			case FloatObject:
				return BooleanObject(i.floatValue() < float64(arg))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: "<=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			i := receiver.(IntegerObject)

			switch rightObject := args[0].(type) {
			case IntegerObject:
				return BooleanObject(int(i) <= int(rightObject))
			case FloatObject:
				return BooleanObject(i.floatValue() <= float64(rightObject))
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name: "float",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return FloatObject(float64(receiver.(IntegerObject)))
		},
		Primitive: true,
	},
	{
		Name: "int",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return receiver
		},
		Primitive: true,
	},
	{
		Name: "string",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			intObj := receiver.(IntegerObject)
			return StringObject(strconv.Itoa(int(intObj)))
		},
		Primitive: true,
	},
	{
		Name: "times",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				// Returns a range object from zero to self - 1
				return initRangeObject(t.vm, 0, int(receiver.(IntegerObject)), true)
				//return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			n := int(receiver.(IntegerObject))
			for i := 0; i < n; i++ {
				t.Yield(blockFrame, IntegerObject(i))
			}
			return receiver
		},
	},
	{
		Name: "json",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return StringObject(receiver.(IntegerObject).ToJSON(t))
		},
	},
}

func initIntegerClass(vm *VM) *RClass {
	intClass = vm.InitClass(classes.IntegerClass).
		ClassMethods(integerClassMethods).
		InstanceMethods(integerInstanceMethods).
		SetConstant("MAX_INT", IntegerObject(math.MaxInt64)).
		SetConstant("MIN_INT", IntegerObject(math.MinInt64))
	return intClass
}

// Value returns the object
func (i IntegerObject) Value() interface{} {
	return int(i)
}

// Numeric interface
func (i IntegerObject) floatValue() float64 {
	return float64(i)
}

// EqualTo apply an equality test, returning true if the objects are considered equal,
// and false otherwise.
func (i IntegerObject) EqualTo(rightObject Object) bool {
	switch rightObject := rightObject.(type) {
	case IntegerObject:
		leftValue := int(i)
		rightValue := int(rightObject)
		return leftValue == rightValue

	case FloatObject:
		leftValue := i.floatValue()
		rightValue := float64(rightObject)
		return leftValue == rightValue

	default:
		return false
	}
}

// ToString returns the object's name as the string format
func (i IntegerObject) ToString(t *Thread) string {
	return strconv.Itoa(int(i))
}

// Inspect delegates to ToString
func (i IntegerObject) Inspect(t *Thread) string {
	return i.ToString(t)
}

// ToJSON just delegates to ToString
func (i IntegerObject) ToJSON(t *Thread) string {
	return i.ToString(t)
}

// equal checks if the integer values between receiver and argument are equal
func (i IntegerObject) equal(e IntegerObject) bool {
	return int(i) == int(e)
}

func (i IntegerObject) lessThan(arg Object) bool {
	switch rightObject := arg.(type) {
	case IntegerObject:
		return int(i) < int(rightObject)
	case FloatObject:
		return i.floatValue() < float64(rightObject)
	}
	return false
}

// Class ...
func (i IntegerObject) Class() *RClass {
	return intClass
}

// GetVariable ...
func (i IntegerObject) GetVariable(string) (Object, bool) {
	return nil, false
}

// SetVariable ...
func (i IntegerObject) SetVariable(n string, o Object) Object {
	return o
}

// FindLookup ...
func (i IntegerObject) FindLookup(searchAncestor bool) (method Object) {
	method, _ = i.Class().Methods[lookupMethod]

	if method == nil && searchAncestor {
		method = i.FindMethod(lookupMethod, false)
	}

	return
}

// FindMethod ...
func (i IntegerObject) FindMethod(methodName string, super bool) Object {
	if super {
		return i.Class().superClass.lookupMethod(methodName)
	}
	return i.Class().lookupMethod(methodName)
}

// Variables ...
func (i IntegerObject) Variables() Environment {
	return nil
}

// SetVariables ...
func (i IntegerObject) SetVariables(Environment) {
}

// IsTruthy ...
func (i IntegerObject) IsTruthy() bool {
	return int(i) != 0
}
