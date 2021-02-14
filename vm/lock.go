package vm

import (
	"sync"

	"github.com/robotii/lito/vm/errors"
)

// LockObject is a Golang Lock.
type LockObject struct {
	BaseObj
	locked bool
	mutex  *sync.Mutex
}

var lockClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return initLockObject(t.vm)
		},
	},
}

var lockInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "lock",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			lockObject := receiver.(*LockObject)
			lockObject.mutex.Lock()
			lockObject.locked = true

			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return receiver
			}
			blockReturnValue := t.Yield(blockFrame)
			lockObject.locked = false
			lockObject.mutex.Unlock()
			return blockReturnValue
		},
	},
	{
		Name: "unlock",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			lockObject := receiver.(*LockObject)
			if lockObject.locked {
				lockObject.locked = false
				lockObject.mutex.Unlock()
			} else {
				return t.vm.InitErrorObject(t, errors.InternalError, "Trying to unlock already unlocked lock")
			}
			return receiver
		},
	},
}

func initLockObject(vm *VM) *LockObject {
	lockClass := vm.loadConstant("Lock", false)
	return &LockObject{
		BaseObj: BaseObj{class: lockClass},
		mutex:   &sync.Mutex{},
	}
}

func initLockClass(vm *VM) {
	vm.objectClass.SetClassConstant(vm.InitClass("Lock").
		ClassMethods(lockClassMethods).
		InstanceMethods(lockInstanceMethods))
}

// Value returns the object
func (lock *LockObject) Value() interface{} {
	return lock.mutex
}

// ToString returns the object's name as the string format
func (lock *LockObject) ToString(t *Thread) string {
	return "#<" + lock.class.Name + " >"
}

// Inspect delegates to ToString
func (lock *LockObject) Inspect(t *Thread) string {
	return lock.ToString(t)
}

// ToJSON just delegates to ToString
func (lock *LockObject) ToJSON(t *Thread) string {
	return lock.ToString(t)
}
