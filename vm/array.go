package vm

import (
	"sort"
	"strings"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// ArrayObject represents an instance of Array class.
type ArrayObject struct {
	BaseObj
	Elements []Object
	splat    bool
}

var arrayClass *RClass

var arrayClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			// Copy the args and make an array from them
			values := make([]Object, len(args))
			copy(values, args)
			return InitArrayObject(values)
		},
		Primitive: true,
	},
}

var arrayInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "[]",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			arr := receiver.(*ArrayObject)
			if len(args) == 1 {
				// TODO: Refactor this as it is ugly
				if rangeObj, ok := args[0].(*RangeObject); ok {
					start := rangeObj.Start
					end := rangeObj.End
					// Handle exclusive ranges
					if rangeObj.Exclusive {
						end--
					}

					if end < 0 {
						end += arr.Len()
					}
					if start < 0 {
						start += arr.Len()
					}
					if start < 0 || start > arr.Len() || end < 0 || end > arr.Len() {
						return NIL
					}
					count := end - start
					if count < 0 {
						return NIL
					}
					args = []Object{IntegerObject(start), IntegerObject(count)}
				}
			}
			return arr.index(t, args)
		},
	},
	{
		Name: "*",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			copiesNumber, ok := args[0].(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
			}

			arr := receiver.(*ArrayObject)
			return arr.times(t, int(copiesNumber))
		},
	},
	{
		Name: "+",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			otherArray, ok := args[0].(*ArrayObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.ArrayClass, args[0].Class().Name)
			}

			selfArray := receiver.(*ArrayObject)
			newArrayElements := append(selfArray.Elements, otherArray.Elements...)
			return InitArrayObject(newArrayElements)
		},
		Primitive: true,
	},
	{
		Name: "[]=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			aLen := len(args)
			if aLen < 2 || aLen > 3 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentRange, 2, 3, aLen)
			}

			i := args[0]
			index, ok := i.(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
			}

			indexValue := int(index)
			arr := receiver.(*ArrayObject)
			if aLen == 3 {
				if indexValue < 0 {
					if arr.normalizeIndex(index) == -1 {
						return t.vm.InitErrorObject(t, errors.ArgumentError, errors.TooSmallIndexValue, indexValue, -arr.Len())
					}
					indexValue = arr.normalizeIndex(index)
				}

				c := args[1]
				count, ok := c.(IntegerObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[1].Class().Name)
				}

				countValue := int(count)
				if countValue < 0 {
					return t.vm.InitErrorObject(t, errors.ArgumentError, errors.NegativeSecondValue, int(count))
				}

				a := args[2]
				assignedValue, isArray := a.(*ArrayObject)

				if indexValue >= arr.Len() {

					for arr.Len() < indexValue {
						arr.Elements = append(arr.Elements, NIL)
					}

					if isArray {
						arr.Elements = append(arr.Elements, assignedValue.Elements...)
					} else {
						arr.Elements = append(arr.Elements, a)
					}
					return a
				}

				endValue := indexValue + countValue
				if endValue > arr.Len() {
					endValue = arr.Len()
				}

				arr.Elements = append(arr.Elements[:indexValue], arr.Elements[endValue:]...)

				if isArray {
					arr.Elements = append(arr.Elements[:indexValue], append(assignedValue.Elements, arr.Elements[indexValue:]...)...)
				} else {
					arr.Elements = append(arr.Elements[:indexValue], append([]Object{a}, arr.Elements[indexValue:]...)...)
				}

				return a
			}
			if indexValue < 0 {
				if len(arr.Elements) < -indexValue {
					return t.vm.InitErrorObject(t, errors.ArgumentError, errors.TooSmallIndexValue, indexValue, -arr.Len())
				}
				arr.Elements[len(arr.Elements)+indexValue] = args[1]
				return arr.Elements[len(arr.Elements)+indexValue]
			}

			// Expand the array
			if len(arr.Elements) < (indexValue + 1) {
				newArr := make([]Object, indexValue+1)
				copy(newArr, arr.Elements)
				for i := len(arr.Elements); i <= indexValue; i++ {
					newArr[i] = NIL
				}
				arr.Elements = newArr
			}

			arr.Elements[indexValue] = args[1]

			return arr.Elements[indexValue]
		},
	},
	{
		Name: "any?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			arr := receiver.(*ArrayObject)
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			if blockFrame.IsEmpty() {
				return FALSE
			}

			for _, obj := range arr.Elements {
				result := t.Yield(blockFrame, obj)
				if result.IsTruthy() {
					return TRUE
				}
			}
			return FALSE
		},
	},
	{
		Name: "include?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			arr := receiver.(*ArrayObject)
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			for _, obj := range arr.Elements {
				if obj.EqualTo(args[0]) {
					return TRUE
				}
			}
			return FALSE
		},
		Primitive: true,
	},
	{
		Name: "clear",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			arr := receiver.(*ArrayObject)
			arr.Elements = []Object{}

			return arr
		},
		Primitive: true,
	},
	{
		Name: "concat",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			arr := receiver.(*ArrayObject)

			for _, arg := range args {
				addAr, ok := arg.(*ArrayObject)

				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.ArrayClass, arg.Class().Name)
				}

				for _, el := range addAr.Elements {
					arr.Elements = append(arr.Elements, el)
				}
			}
			return arr
		},
		Primitive: true,
	},
	{
		Name: "count",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			aLen := len(args)
			if aLen > 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 1, aLen)
			}

			arr := receiver.(*ArrayObject)
			var count int
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				if aLen == 0 {
					return IntegerObject(len(arr.Elements))
				}

				// TODO: This is a mess, clean it up
				arg := args[0]
				findInt, findIsInt := arg.(IntegerObject)
				findString, findIsString := arg.(StringObject)
				findBoolean, findIsBoolean := arg.(BooleanObject)

				for i := 0; i < len(arr.Elements); i++ {
					el := arr.Elements[i]
					switch el := el.(type) {
					case IntegerObject:
						if findIsInt && findInt.equal(el) {
							count++
						}
					case StringObject:
						if findIsString && findString.equal(el) {
							count++
						}
					case BooleanObject:
						if findIsBoolean && findBoolean.equal(el) {
							count++
						}
					}
				}

				return IntegerObject(count)
			}

			// Block was given
			if blockFrame.IsEmpty() {
				return IntegerObject(0)
			}

			for _, obj := range arr.Elements {
				result := t.Yield(blockFrame, obj)
				if result.IsTruthy() {
					count++
				}
			}

			return IntegerObject(count)
		},
	},
	{
		Name: "delete_at",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			i := args[0]
			index, ok := i.(IntegerObject)

			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
			}

			arr := receiver.(*ArrayObject)
			normalizedIndex := arr.normalizeIndex(index)

			if normalizedIndex == -1 {
				return NIL
			}

			// delete and slice
			deletedValue := arr.Elements[normalizedIndex]
			arr.Elements = append(arr.Elements[:normalizedIndex], arr.Elements[normalizedIndex+1:]...)
			return deletedValue
		},
	},
	{
		Name: "dup",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			arr, _ := receiver.(*ArrayObject)
			newArr := make([]Object, len(arr.Elements))
			copy(newArr, arr.Elements)
			newObj := InitArrayObject(newArr)
			newObj.SetVariables(arr.Variables().copy())
			return newObj
		},
		Primitive: true,
	},
	{
		Name: "each",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}
			if blockFrame.IsEmpty() {
				return receiver
			}

			arr := receiver.(*ArrayObject)

			for _, obj := range arr.Elements {
				t.Yield(blockFrame, obj)
				if blockFrame.IsRemoved() {
					break
				}
			}
			return arr
		},
	},
	{
		Name: "each_index",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			arr := receiver.(*ArrayObject)
			if blockFrame.IsEmpty() {
				return arr
			}

			for i := range arr.Elements {
				t.Yield(blockFrame, IntegerObject(i))
			}
			return arr
		},
	},
	{
		Name: "empty?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return BooleanObject(receiver.(*ArrayObject).Len() == 0)
		},
		Primitive: true,
	},
	{
		Name: "first",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			// TODO: Support using block predicate here
			aLen := len(args)
			if aLen > 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 1, aLen)
			}

			arr := receiver.(*ArrayObject)
			arrLength := len(arr.Elements)
			if arrLength == 0 {
				return NIL
			}

			if aLen == 0 {
				return arr.Elements[0]
			}

			arg, ok := args[0].(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
			}

			if int(arg) < 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.NegativeValue, int(arg))
			}

			if arrLength > int(arg) {
				return InitArrayObject(arr.Elements[:int(arg)])
			}
			return arr
		},
	},
	{
		Name: "flatten",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return InitArrayObject(receiver.(*ArrayObject).flatten())
		},
	},
	{
		Name: "index_with",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			a := receiver.(*ArrayObject)

			hash := make(map[string]Object)
			switch len(args) {
			case 0:
				for _, obj := range a.Elements {
					hash[obj.ToString(t)] = t.Yield(blockFrame, obj)
				}
			case 1:
				arg := args[0]
				for _, obj := range a.Elements {
					switch b := t.Yield(blockFrame, obj); b.(type) {
					case *NilObject:
						hash[obj.ToString(t)] = arg
					default:
						hash[obj.ToString(t)] = b
					}
				}
			default:
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 1, len(args))
			}

			return InitHashObject(hash)
		},
	},
	{
		Name: "join",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			aLen := len(args)
			if aLen < 0 || aLen > 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentRange, 0, 1, aLen)
			}

			var sep string
			if aLen == 0 {
				sep = ""
			} else {
				arg, ok := args[0].(StringObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
				}
				sep = string(arg)
			}

			arr := receiver.(*ArrayObject)
			var elements []string
			for _, e := range arr.flatten() {
				elements = append(elements, e.ToString(t))
			}

			return StringObject(strings.Join(elements, sep))
		},
	},
	{
		Name: "last",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			aLen := len(args)
			if aLen > 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 1, aLen)
			}

			arr := receiver.(*ArrayObject)
			arrLength := len(arr.Elements)

			if aLen == 0 {
				return arr.Elements[arrLength-1]
			}

			arg, ok := args[0].(IntegerObject)

			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
			}

			if int(arg) < 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.NegativeValue, int(arg))
			}

			if arrLength > int(arg) {
				return InitArrayObject(arr.Elements[arrLength-int(arg) : arrLength])
			}
			return arr
		},
	},
	{
		Name: "length",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			arr := receiver.(*ArrayObject)
			return IntegerObject(arr.Len())
		},
	},
	{
		Name: "map",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			arr := receiver.(*ArrayObject)
			var elements = make([]Object, len(arr.Elements))
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			if blockFrame.IsEmpty() {
				for i := 0; i < len(arr.Elements); i++ {
					elements[i] = NIL
				}
			} else {
				for i, obj := range arr.Elements {
					result := t.Yield(blockFrame, obj)
					elements[i] = result
				}
			}

			return InitArrayObject(elements)
		},
	},
	{
		Name: "pop",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			arr := receiver.(*ArrayObject)
			return arr.pop()
		},
	},
	{
		Name: "push",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			arr := receiver.(*ArrayObject)
			return arr.push(args)
		},
	},
	{
		Name: "reduce",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			aLen := len(args)
			if aLen > 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 1, aLen)
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			arr := receiver.(*ArrayObject)

			if blockFrame.IsEmpty() {
				return NIL
			}

			var prev Object
			var start int
			switch aLen {
			case 0:
				prev = arr.Elements[0]
				start = 1
			case 1:
				prev = args[0]
				start = 0
			}

			for i := start; i < len(arr.Elements); i++ {
				result := t.Yield(blockFrame, prev, arr.Elements[i])
				prev = result
			}

			return prev
		},
	},
	{
		Name: "reverse",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			arr := receiver.(*ArrayObject)
			return arr.reverse()
		},
		Primitive: true,
	},
	{
		Name: "reverse_each",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			arr := receiver.(*ArrayObject)
			reversedArr := arr.reverse()
			if blockFrame.IsEmpty() {
				return reversedArr
			}

			for _, obj := range reversedArr.Elements {
				t.Yield(blockFrame, obj)
			}

			return reversedArr
		},
	},
	{
		Name: "filter",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			arr := receiver.(*ArrayObject)
			var elements []Object
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			if blockFrame.IsEmpty() {
				return InitArrayObject(elements)
			}

			for _, obj := range arr.Elements {
				result := t.Yield(blockFrame, obj)
				if result.IsTruthy() {
					elements = append(elements, obj)
				}
			}

			return InitArrayObject(elements)
		},
	},
	{
		Name: "shift",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			arr := receiver.(*ArrayObject)
			return arr.shift()
		},
	},
	{
		Name: "sort",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, "Expect 0 argument. got=%d", len(args))
			}

			arr := receiver.(*ArrayObject)
			newArr := arr.copy().(*ArrayObject)
			sort.Sort(newArr)
			return newArr
		},
	},
	{
		Name: "hash",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			ary := receiver.(*ArrayObject)

			hash := make(map[string]Object)
			for i, el := range ary.Elements {
				kv, ok := el.(*ArrayObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, "Expect the Array's element #%d to be Array. got: %s", i, el.Class().Name)
				}

				if len(kv.Elements) != 2 {
					return t.vm.InitErrorObject(t, errors.ArgumentError, "Expect element #%d to have 2 elements as a key-value pair. got: %s", i, kv.ToString(t))
				}

				k := kv.Elements[0]
				if _, ok := k.(StringObject); !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, "Expect the key in the Array's element #%d to be String. got: %s", i, k.Class().Name)
				}

				hash[k.ToString(t)] = kv.Elements[1]
			}

			return InitHashObject(hash)
		},
	},
	{
		Name: "json",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return StringObject(receiver.(*ArrayObject).ToJSON(t))
		},
	},
	{
		Name: "unshift",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return receiver.(*ArrayObject).unshift(args)
		},
	},
	{
		Name: "values",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			arr := receiver.(*ArrayObject)
			var elements = make([]Object, len(args))

			for i, arg := range args {
				index, ok := arg.(IntegerObject)

				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, arg.Class().Name)
				}

				if int(index) >= len(arr.Elements) {
					elements[i] = NIL
				} else if int(index) < 0 && -int(index) > len(arr.Elements) {
					elements[i] = NIL
				} else if int(index) < 0 {
					elements[i] = arr.Elements[len(arr.Elements)+int(index)]
				} else {
					elements[i] = arr.Elements[int(index)]
				}
			}

			return InitArrayObject(elements)
		},
	},
}

// InitArrayObject returns a new object with the given elemnts
func InitArrayObject(elements []Object) *ArrayObject {
	return &ArrayObject{
		BaseObj:  BaseObj{class: arrayClass},
		Elements: elements,
	}
}

func initArrayClass(vm *VM) *RClass {
	arrayClass = vm.InitClass(classes.ArrayClass).
		ClassMethods(arrayClassMethods).
		InstanceMethods(arrayInstanceMethods)
	return arrayClass
}

// Value returns the elements from the object
func (a *ArrayObject) Value() interface{} {
	return a.Elements
}

// ToString returns the object's elements as the string format
func (a *ArrayObject) ToString(t *Thread) string {
	var out strings.Builder

	var elements []string
	for _, e := range a.Elements {
		elements = append(elements, e.Inspect(t))
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// Inspect delegates to ToString
func (a *ArrayObject) Inspect(t *Thread) string {
	return a.ToString(t)
}

// ToJSON returns the object's elements as the JSON string format
func (a *ArrayObject) ToJSON(t *Thread) string {
	var out strings.Builder
	elements := []string{}
	for _, e := range a.Elements {
		elements = append(elements, e.ToJSON(t))
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// times returns a array composed of N copies of the array
func (a *ArrayObject) times(t *Thread, n int) Object {
	aLen := len(a.Elements)
	result := make([]Object, 0, aLen*n)

	for i := 0; i < n; i++ {
		result = append(result, a.Elements...)
	}

	return InitArrayObject(result)
}

// Retrieves an object in an array using Integer index.
func (a *ArrayObject) index(t *Thread, args []Object) Object {
	aLen := len(args)
	if aLen < 1 || aLen > 2 {
		return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentRange, 1, 2, aLen)
	}

	i := args[0]
	index, ok := i.(IntegerObject)
	arrLength := a.Len()

	if !ok {
		return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
	}

	if int(index) < 0 && int(index) < -arrLength {
		return t.vm.InitErrorObject(t, errors.ArgumentError, errors.TooSmallIndexValue, int(index), -arrLength)
	}

	// Validation for the second argument if exists
	if aLen == 2 {
		j := args[1]
		count, ok := j.(IntegerObject)

		if !ok {
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[1].Class().Name)
		}
		if int(count) < 0 {
			return t.vm.InitErrorObject(t, errors.ArgumentError,
				errors.NegativeSecondValue, int(count))
		}
		if int(index) > 0 && int(index) == arrLength {
			return InitArrayObject([]Object{})
		}
	}

	// Start Indexing
	normalizedIndex := a.normalizeIndex(index)
	if normalizedIndex == -1 {
		return NIL
	}

	if aLen == 2 {
		j := args[1]
		count, _ := j.(IntegerObject)

		if normalizedIndex+int(count) > arrLength {
			return InitArrayObject(a.Elements[normalizedIndex:])
		}
		return InitArrayObject(a.Elements[normalizedIndex : normalizedIndex+int(count)])
	}

	return a.Elements[normalizedIndex]
}

// flatten returns a array of Objects that is one-dimensional flattening of Elements
func (a *ArrayObject) flatten() []Object {
	var result []Object

	for _, e := range a.Elements {
		if arr, isArray := e.(*ArrayObject); isArray {
			result = append(result, arr.flatten()...)
		} else {
			result = append(result, e)
		}
	}

	return result
}

// Len returns the length of array's elements
func (a *ArrayObject) Len() int {
	return len(a.Elements)
}

// Swap is one of the required method to fulfill sortable interface
func (a *ArrayObject) Swap(i, j int) {
	a.Elements[i], a.Elements[j] = a.Elements[j], a.Elements[i]
}

// Less is one of the required method to fulfill sortable interface
func (a *ArrayObject) Less(i, j int) bool {
	leftObj, rightObj := a.Elements[i], a.Elements[j]
	switch leftObj := leftObj.(type) {
	case Numeric:
		return leftObj.lessThan(rightObj)
	case StringObject:
		right, ok := rightObj.(StringObject)
		return ok && string(leftObj) < string(right)
	default:
		return false
	}
}

func (a *ArrayObject) normalizeIndex(objectIndex IntegerObject) int {
	aLength := len(a.Elements)
	index := int(objectIndex)

	// out of bounds
	if index >= aLength {
		return -1
	}
	if index < 0 && -index > aLength {
		return -1
	}

	// within bounds
	if index < 0 {
		return aLength + index
	}

	return index
}

// pop removes the last element in the array and returns it
func (a *ArrayObject) pop() Object {
	if len(a.Elements) < 1 {
		return NIL
	}

	value := a.Elements[len(a.Elements)-1]
	a.Elements = a.Elements[:len(a.Elements)-1]
	return value
}

// push appends given object into array and returns the array object
func (a *ArrayObject) push(objs []Object) *ArrayObject {
	a.Elements = append(a.Elements, objs...)
	return a
}

// returns a reversed copy of the passed array
func (a *ArrayObject) reverse() *ArrayObject {
	arrLen := len(a.Elements)
	reversedArrElems := make([]Object, arrLen)

	for i, element := range a.Elements {
		reversedArrElems[arrLen-i-1] = element
	}

	return &ArrayObject{
		BaseObj:  BaseObj{class: a.class},
		Elements: reversedArrElems,
	}
}

// shift removes the first element in the array and returns it
func (a *ArrayObject) shift() Object {
	if len(a.Elements) < 1 {
		return NIL
	}

	value := a.Elements[0]
	a.Elements = a.Elements[1:]
	return value
}

// copy returns the duplicate of the Array object
func (a *ArrayObject) copy() Object {
	e := make([]Object, len(a.Elements))

	copy(e, a.Elements)

	return &ArrayObject{
		BaseObj:  BaseObj{class: a.class},
		Elements: e,
	}
}

// EqualTo returns if the ArrayObject is equal to another object
func (a *ArrayObject) EqualTo(compared Object) bool {
	c, ok := compared.(*ArrayObject)
	if !ok {
		return false
	}

	if len(a.Elements) != len(c.Elements) {
		return false
	}

	for i, e := range a.Elements {
		if !e.EqualTo(c.Elements[i]) {
			return false
		}
	}

	return true
}

// unshift inserts an element in the first position of the array
func (a *ArrayObject) unshift(objs []Object) *ArrayObject {
	// Make sure we create a new slice
	o := make([]Object, 0, len(a.Elements)+len(objs))
	o = append(o, objs...)
	o = append(o, a.Elements...)
	a.Elements = o
	return a
}
