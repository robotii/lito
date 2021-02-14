package vm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// HashObject represents a map instance.
// TODO: Think about a new representation for this
type HashObject struct {
	BaseObj
	Pairs map[string]Object
}

var hashClass *RClass

var hashClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return InitHashObject(make(map[string]Object))
		},
		Primitive: true,
	},
}

var hashInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "[]",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			key, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError,
					errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			h := receiver.(*HashObject)

			value, ok := h.Pairs[string(key)]
			if !ok {
				return NIL
			}

			return value
		},
	},
	{
		Name: "[]=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError,
					errors.WrongNumberOfArgument, 2, len(args))
			}
			k := args[0]
			key, ok := k.(StringObject)

			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError,
					errors.WrongArgumentTypeFormat, classes.StringClass, k.Class().Name)
			}

			h := receiver.(*HashObject)
			h.Pairs[string(key)] = args[1]

			return args[1]
		},
	},
	{
		Name: "any?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			hash := receiver.(*HashObject)
			if blockFrame.IsEmpty() {
				return FALSE
			}

			for stringKey, value := range hash.Pairs {
				objectKey := StringObject(stringKey)
				result := t.Yield(blockFrame, objectKey, value)

				if blockFrame.IsRemoved() {
					return NIL
				}

				if result.IsTruthy() {
					return TRUE
				}
			}

			return FALSE
		},
	},
	{
		Name: "clear",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			h := receiver.(*HashObject)
			h.Pairs = make(map[string]Object)
			return h
		},
	},
	{
		Name: "delete",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			hash := receiver.(*HashObject)
			for _, d := range args {
				deleteKey, ok := d.(StringObject)
				if ok {
					delete(hash.Pairs, string(deleteKey))
				} else {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, d.Class().Name)
				}
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil || blockFrame.IsEmpty() {
				return hash
			}

			for stringKey, value := range hash.Pairs {
				objectKey := StringObject(stringKey)
				result := t.Yield(blockFrame, objectKey, value)

				booleanResult, isResultBoolean := result.(BooleanObject)

				if isResultBoolean {
					if booleanResult {
						delete(hash.Pairs, stringKey)
					}
				} else if result != NIL {
					delete(hash.Pairs, stringKey)
				}
			}

			return hash
		},
	},
	{
		Name: "dup",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return receiver.(*HashObject).copy()
		},
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

			h := receiver.(*HashObject)

			keys := h.sortedKeys()

			for _, k := range keys {
				v := h.Pairs[k]
				strK := StringObject(k)

				t.Yield(blockFrame, strK, v)

				// If we break inside the block, then stop the iteration
				if blockFrame.IsRemoved() {
					break
				}
			}

			return h
		},
	},
	{
		Name: "each_key",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			h := receiver.(*HashObject)

			keys := h.sortedKeys()
			var arrOfKeys []Object

			for _, k := range keys {
				obj := StringObject(k)
				arrOfKeys = append(arrOfKeys, obj)
				t.Yield(blockFrame, obj)
			}

			return InitArrayObject(arrOfKeys)
		},
	},
	{
		Name: "each_value",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			h := receiver.(*HashObject)

			keys := h.sortedKeys()
			var arrOfValues []Object

			for _, k := range keys {
				value := h.Pairs[k]
				arrOfValues = append(arrOfValues, value)
				t.Yield(blockFrame, value)
			}

			return InitArrayObject(arrOfValues)
		},
	},
	{
		Name: "empty?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			h := receiver.(*HashObject)
			return BooleanObject(h.length() == 0)
		},
	},
	{
		Name: "key?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			h := receiver.(*HashObject)
			input, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			_, ok = h.Pairs[string(input)]
			return BooleanObject(ok)
		},
	},
	{
		Name: "value?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			h := receiver.(*HashObject)
			for _, v := range h.Pairs {
				if v.EqualTo(args[0]) {
					return TRUE
				}
			}
			return FALSE
		},
	},
	{
		Name: "length",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			h := receiver.(*HashObject)
			return IntegerObject(h.length())
		},
	},
	{
		Name: "map_values",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			h := receiver.(*HashObject)
			if blockFrame.IsEmpty() {
				return h
			}

			resultHash := make(map[string]Object)
			for k, v := range h.Pairs {
				result := t.Yield(blockFrame, v)
				resultHash[k] = result
			}
			return InitHashObject(resultHash)
		},
	},
	{
		Name: "merge",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) < 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentMore, 1, len(args))
			}

			h := receiver.(*HashObject)
			result := make(map[string]Object)
			for k, v := range h.Pairs {
				result[k] = v
			}

			for _, obj := range args {
				hashObj, ok := obj.(*HashObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.HashClass, obj.Class().Name)
				}
				for k, v := range hashObj.Pairs {
					result[k] = v
				}
			}

			return InitHashObject(result)
		},
	},
	{
		Name: "filter",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			destinationPairs := map[string]Object{}
			if blockFrame.IsEmpty() {
				return InitHashObject(destinationPairs)
			}

			sourceHash := receiver.(*HashObject)

			for stringKey, value := range sourceHash.Pairs {
				objectKey := StringObject(stringKey)
				result := t.Yield(blockFrame, objectKey, value)

				if result.IsTruthy() {
					destinationPairs[stringKey] = value
				}
			}

			return InitHashObject(destinationPairs)
		},
	},
	{
		Name: "keys",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			h := receiver.(*HashObject)
			sortedKeys := h.sortedKeys()
			var keys []Object
			for _, k := range sortedKeys {
				keys = append(keys, StringObject(k))
			}
			return InitArrayObject(keys)
		},
	},
	{
		Name: "array",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			aLen := len(args)
			if aLen != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, aLen)
			}

			h := receiver.(*HashObject)
			var resultArr []Object
			for _, k := range h.sortedKeys() {
				var pairArr []Object
				pairArr = append(pairArr, StringObject(k))
				pairArr = append(pairArr, h.Pairs[k])
				resultArr = append(resultArr, InitArrayObject(pairArr))
			}
			return InitArrayObject(resultArr)
		},
	},
	{
		Name: "json",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			r := receiver.(*HashObject)
			return StringObject(r.ToJSON(t))
		},
	},
	{
		Name: "string",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			h := receiver.(*HashObject)
			return StringObject(h.ToString(t))
		},
	},
	{
		Name: "values",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				hash := receiver.(*HashObject)
				var result []Object

				for _, objectKey := range args {
					stringObjectKey, ok := objectKey.(StringObject)

					if !ok {
						return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, objectKey.Class().Name)
					}

					value, ok := hash.Pairs[string(stringObjectKey)]

					if !ok {
						value = NIL
					}

					result = append(result, value)
				}

				return InitArrayObject(result)
			}

			h := receiver.(*HashObject)
			var keys []Object
			sortedKeys := h.sortedKeys()
			for _, v := range sortedKeys {
				keys = append(keys, h.Pairs[v])
			}
			return InitArrayObject(keys)
		},
	},
}

// InitHashObject initialise the HashObject
func InitHashObject(pairs map[string]Object) *HashObject {
	return &HashObject{
		BaseObj: BaseObj{class: hashClass},
		Pairs:   pairs,
	}
}

func initHashClass(vm *VM) *RClass {
	hashClass = vm.InitClass(classes.HashClass).
		ClassMethods(hashClassMethods).
		InstanceMethods(hashInstanceMethods)
	return hashClass
}

// Value returns the object
func (h *HashObject) Value() interface{} {
	return h.Pairs
}

// ToString returns the object's name as the string format
func (h *HashObject) ToString(t *Thread) string {
	var out strings.Builder
	var pairs []string

	for _, key := range h.sortedKeys() {
		pairs = append(pairs, fmt.Sprintf("%s: %s", key, h.Pairs[key].Inspect(t)))
	}

	out.WriteString("{ ")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString(" }")

	return out.String()
}

// Inspect delegates to ToString
func (h *HashObject) Inspect(t *Thread) string {
	return h.ToString(t)
}

// ToJSON returns the object's name as the JSON string format
func (h *HashObject) ToJSON(t *Thread) string {
	var out strings.Builder
	var values []string
	pairs := h.Pairs
	out.WriteString("{")

	for key, value := range pairs {
		values = append(values, generateJSONFromPair(key, value, t))
	}

	out.WriteString(strings.Join(values, ","))
	out.WriteString("}")
	return out.String()
}

// Returns the length of the hash
func (h *HashObject) length() int {
	return len(h.Pairs)
}

// Returns the sorted keys of the hash
func (h *HashObject) sortedKeys() []string {
	var arr []string
	for k := range h.Pairs {
		arr = append(arr, k)
	}
	sort.Strings(arr)
	return arr
}

// Returns the duplicate of the Hash object
func (h *HashObject) copy() Object {
	elems := map[string]Object{}

	for k, v := range h.Pairs {
		elems[k] = v
	}

	newHash := &HashObject{
		BaseObj: BaseObj{class: h.class},
		Pairs:   elems,
	}

	return newHash
}

// EqualTo returns true if the HashObject is equal to the given Object
func (h *HashObject) EqualTo(with Object) bool {
	w, ok := with.(*HashObject)
	if !ok {
		return false
	}

	if len(h.Pairs) != len(w.Pairs) {
		return false
	}

	for k, v := range h.Pairs {
		if !v.EqualTo(w.Pairs[k]) {
			return false
		}
	}

	return true
}

// Return the JSON style strings of the Hash object
func generateJSONFromPair(key string, v Object, t *Thread) string {
	var out strings.Builder

	out.WriteString("\"" + key + "\"")
	out.WriteString(":")
	out.WriteString(v.ToJSON(t))

	return out.String()
}
