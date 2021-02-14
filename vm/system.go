package vm

import (
	"os"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

var systemClassMethods = []*BuiltinMethodObject{
	{
		Name:      "new",
		Fn:        NoSuchMethod("new"),
		Primitive: true,
	},
	{
		Name: "exit",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			aLen := len(args)
			switch aLen {
			case 0:
				os.Exit(0)
			case 1:
				exitCode, ok := args[0].(IntegerObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
				}

				os.Exit(int(exitCode))
			default:
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 1, aLen)
			}
			return NIL // Compiler requires this apparently
		},
	},
}

func initSystemClass(vm *VM) *RClass {
	return vm.InitClass(classes.SystemClass).
		ClassMethods(systemClassMethods)
}
