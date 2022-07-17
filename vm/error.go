package vm

import (
	"fmt"

	"strings"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// Error class is a struct to hold internal error types with messages.
type Error struct {
	BaseObj
	message      string
	stackTraces  []string
	storedTraces bool
	Type         string
	Ignore       bool
	Raised       bool
}

var errorClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) > 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 1, len(args))
			}
			if r, ok := receiver.(*RClass); ok {
				message := ""
				if len(args) == 1 {
					if str, ok := args[0].(StringObject); ok {
						message = string(str)
					} else {
						return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
					}
				}
				e := t.vm.InitErrorObject(t, r.Name, message)
				e.Ignore = true
				e.Raised = false
				return e
			}
			return NIL
		},
	},
}

var errorInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "message",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if r, ok := receiver.(*Error); ok {
				return StringObject(r.message)
			}
			return StringObject("")
		},
	},
	{
		Name: "stack",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if r, ok := receiver.(*Error); ok {
				return StringObject(strings.Join(r.stackTraces, "\n"))
			}
			return StringObject("")
		},
	},
	{
		Name: "type",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if r, ok := receiver.(*Error); ok {
				return StringObject(r.Type)
			}
			return StringObject("")
		},
	},
	{
		Name: "raise",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if r, ok := receiver.(*Error); ok {
				r.Raised = true
			}
			return receiver
		},
	},
	{
		Name: "ignore",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if r, ok := receiver.(*Error); ok {
				r.Ignore = true
			}
			return receiver
		},
	},
	{
		Name: "cancel",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if r, ok := receiver.(*Error); ok {
				r.Raised = false
				r.Ignore = false
			}
			return receiver
		},
	},
}

// InitNoMethodError is to print unsupported method errors. This is exported for using from sub-packages.
func (vm *VM) InitNoMethodError(t *Thread, methodName string, receiver Object) *Error {
	return vm.InitErrorObject(t, errors.NoMethodError, errors.UndefinedMethod, methodName, receiver.Inspect(t))
}

// InitErrorObject is to initialise and return an error object.
func (vm *VM) InitErrorObject(t *Thread, errorType string, format string, args ...interface{}) *Error {
	errClass := vm.objectClass.getClassConstant(errorType)
	cf := t.callFrameStack.top()
	sourceLine := t.GetSourceLine()

	switch cfTmp := cf.(type) {
	case *CallFrame:
		// If program counter is 0 means we need to trace back to previous call frame
		if cfTmp.pc == 0 {
			t.callFrameStack.pop()
			cf, _ = t.callFrameStack.top().(*CallFrame)
		}
	}

	return &Error{
		BaseObj:     BaseObj{class: errClass},
		message:     fmt.Sprintf(errorType+": "+format, args...),
		stackTraces: []string{fmt.Sprintf("from %s:%d", cf.FileName(), sourceLine)},
		Type:        errorType,
		Raised:      true,
	}
}

func initErrorClasses(vm *VM) {
	// Create a root Error class
	vm.errorClass = vm.InitClass(classes.ErrorClass).
		InstanceMethods(errorInstanceMethods).
		ClassMethods(errorClassMethods)
	vm.objectClass.SetClassConstant(vm.errorClass)

	// Add in each error class
	for _, errType := range errors.Classes {
		vm.objectClass.SetClassConstant(
			vm.InitClass(errType).inherits(vm.errorClass))
	}
}

func (e *Error) storeStackTraces(t *Thread) {
	// Discard the creation location of this error
	// But keep existing stack trace if we are re-raising this error
	if !e.storedTraces {
		e.stackTraces = nil
	}
	for i := t.callFrameStack.pointer - 1; i >= 0; i-- {
		frame := t.callFrameStack.callFrames[i]

		// TODO: Show go frames as well
		cf, ok := frame.(*CallFrame)
		if ok {
			var sourceLine int
			insCount := cf.instructionsCount()
			// If we encounter an empty block, skip it
			if insCount == 0 {
				continue
			}

			// Work out which line is the best match
			sMap := cf.instructionSet.SourceMap
			if cf.pc >= insCount {
				sourceLine = sMap[insCount-1]
			} else if cf.pc <= 0 {
				sourceLine = sMap[0]
			} else {
				sourceLine = sMap[cf.pc-1]
			}
			msg := fmt.Sprintf("from %s:%d", frame.FileName(), sourceLine)
			e.stackTraces = append(e.stackTraces, msg)
		}
	}

	e.storedTraces = true
}

// ToString returns the object's name as the string format
func (e *Error) ToString(t *Thread) string {
	return e.message
}

// Inspect delegates to ToString
func (e *Error) Inspect(t *Thread) string {
	return e.ToString(t)
}

// ToJSON just delegates to `ToString`
func (e *Error) ToJSON(t *Thread) string {
	return e.ToString(t)
}

// Value returns the message associated with the error
func (e *Error) Value() interface{} {
	return e.message
}

// Message prints the error's message and its stack traces
func (e *Error) Message() string {
	return e.message + "\n" + strings.Join(e.stackTraces, "\n")
}
