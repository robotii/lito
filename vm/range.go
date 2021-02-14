package vm

import (
	"fmt"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// RangeObject is the built in range class
// Range represents an interval: a set of values from the beginning to the end specified.
// Currently, only Integer objects or integer literal are supported.
type RangeObject struct {
	BaseObj
	Start     int
	End       int
	Exclusive bool
}

var rangeClassMethods = []*BuiltinMethodObject{
	{
		Name:      "new",
		Fn:        NoSuchMethod("new"),
		Primitive: true,
	},
}

var rangeInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "each",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			ro := receiver.(*RangeObject)

			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			return ro.each(func(i int) {
				t.Yield(blockFrame, IntegerObject(i))
			})
		},
	},
	{
		Name: "first",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return IntegerObject(receiver.(*RangeObject).Start)
		},
		Primitive: true,
	},
	{
		Name: "last",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return IntegerObject(receiver.(*RangeObject).End)
		},
		Primitive: true,
	},
	{
		Name: "exclusive?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return BooleanObject(receiver.(*RangeObject).Exclusive)
		},
		Primitive: true,
	},
	{
		Name: "map",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			ro := receiver.(*RangeObject)
			el := make([]Object, 0, ro.size())
			if blockFrame.IsEmpty() {
				ro.each(func(i int) {
					el = append(el, NIL)
				})
			} else {
				ro.each(func(i int) {
					el = append(el, t.Yield(blockFrame, IntegerObject(i)))
				})
			}

			return InitArrayObject(el)
		},
	},
	{
		Name: "size",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return IntegerObject(receiver.(*RangeObject).size())
		},
		Primitive: true,
	},
	{
		Name: "step",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			ro := receiver.(*RangeObject)
			step := int(args[0].(IntegerObject))
			if step <= 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.NegativeValue, step)
			}

			return ro.each(func(i int) {
				if (i-ro.Start)%step == 0 {
					t.Yield(blockFrame, IntegerObject(i))
				}
			})
		},
	},
	{
		Name: "array",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			ro := receiver.(*RangeObject)
			el := make([]Object, 0, ro.size())

			ro.each(func(i int) {
				el = append(el, IntegerObject(i))
			})

			return InitArrayObject(el)
		},
		Primitive: true,
	},
	{
		Name: "string",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return StringObject(receiver.(*RangeObject).ToString(t))
		},
		Primitive: true,
	},
}

func initRangeObject(vm *VM, start, end int, exclusive bool) *RangeObject {
	return &RangeObject{
		BaseObj:   BaseObj{class: vm.TopLevelClass(classes.RangeClass)},
		Start:     start,
		End:       end,
		Exclusive: exclusive,
	}
}

func initRangeClass(vm *VM) *RClass {
	return vm.InitClass(classes.RangeClass).
		ClassMethods(rangeClassMethods).
		InstanceMethods(rangeInstanceMethods)
}

// ToString returns the object's name as the string format
func (ro *RangeObject) ToString(t *Thread) string {
	if ro.Exclusive {
		return fmt.Sprintf("(%d...%d)", ro.Start, ro.End)
	}
	return fmt.Sprintf("(%d..%d)", ro.Start, ro.End)
}

// Inspect delegates to ToString
func (ro *RangeObject) Inspect(t *Thread) string {
	return ro.ToString(t)
}

// ToJSON just delegates to ToString
func (ro *RangeObject) ToJSON(t *Thread) string {
	return ro.ToString(t)
}

// Value returns range object's string format
func (ro *RangeObject) Value() interface{} {
	return ro.ToString(nil)
}

func (ro *RangeObject) each(f func(int)) *RangeObject {
	inc := 1
	if ro.End-ro.Start < 0 {
		inc = -1
	}
	end := ro.End + inc
	if ro.Exclusive {
		end = ro.End
	}
	for i := ro.Start; i != end; i += inc {
		f(i)
	}
	return ro
}

func (ro *RangeObject) size() int {
	inc := 1
	if ro.Exclusive {
		inc = 0
	}
	if ro.Start <= ro.End {
		return ro.End - ro.Start + inc
	}
	return ro.Start - ro.End + inc
}

// EqualTo returns if the RangeObject is equal to another object
func (ro *RangeObject) EqualTo(with Object) bool {
	right, ok := with.(*RangeObject)
	return ok &&
		ro.Start == right.Start &&
		ro.End == right.End &&
		ro.Exclusive == right.Exclusive
}
