package vm

import (
	"fmt"
	"sync"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// WaitGroupObject ...
type WaitGroupObject struct {
	BaseObj
	WaitGroup sync.WaitGroup
	// TODO: Add a count that we can measure?
}

func newWaitGroupObject(vm *VM) *WaitGroupObject {
	return &WaitGroupObject{BaseObj: BaseObj{class: vm.TopLevelClass(classes.WaitGroupClass)}, WaitGroup: sync.WaitGroup{}}
}

var waitGroupClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return newWaitGroupObject(t.vm)
		},
		Primitive: true,
	},
}

var waitGroupInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "add",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			wg := receiver.(*WaitGroupObject)
			add, ok := args[0].(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
			}

			wg.WaitGroup.Add(int(add))
			return receiver
		},
		Primitive: true,
	},
	{
		Name: "wait",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			wg := receiver.(*WaitGroupObject)
			wg.WaitGroup.Wait()
			return receiver
		},
		Primitive: true,
	},
	{
		Name: "done",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			wg := receiver.(*WaitGroupObject)
			wg.WaitGroup.Done()
			return receiver
		},
		Primitive: true,
	},
}

func initWaitGroupClass(vm *VM) *RClass {
	return vm.InitClass(classes.WaitGroupClass).
		ClassMethods(waitGroupClassMethods).
		InstanceMethods(waitGroupInstanceMethods)
}

// Value returns the object
func (co *WaitGroupObject) Value() interface{} {
	return &co.WaitGroup
}

// ToString returns the object's name as the string format
func (co *WaitGroupObject) ToString(t *Thread) string {
	return fmt.Sprintf("<WaitGroup: %p>", &co.WaitGroup)
}

// Inspect delegates to ToString
func (co *WaitGroupObject) Inspect(t *Thread) string {
	return co.ToString(t)
}

// ToJSON just delegates to ToString
func (co *WaitGroupObject) ToJSON(t *Thread) string {
	return co.ToString(t)
}

// copy ...
func (co *WaitGroupObject) copy() Object {
	newWg := &WaitGroupObject{BaseObj: BaseObj{class: co.class}, WaitGroup: sync.WaitGroup{}}
	return newWg
}
