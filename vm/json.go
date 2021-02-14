package vm

import (
	"encoding/json"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

var jsonClassMethods = []*BuiltinMethodObject{
	{
		Name:      "new",
		Fn:        NoSuchMethod("new"),
		Primitive: true,
	},
	{
		Name: "parse",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			j, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			var o interface{}
			jsonString := string(j)
			jsonBytes := []byte(jsonString)
			if err := json.Unmarshal(jsonBytes, &o); err != nil {
				return t.vm.InitErrorObject(t, errors.InternalError, "Can't parse string `%s` as json: %s", jsonString, err.Error())
			}
			return t.vm.InitObjectFromGoType(o)
		},
		Primitive: true,
	},
}

var jsonInstanceMethods = []*BuiltinMethodObject{}

func initJSONClass(vm *VM) {
	class := vm.InitClass("JSON").
		ClassMethods(jsonClassMethods).
		InstanceMethods(jsonInstanceMethods)
	vm.TopLevelClass(classes.ObjectClass).SetClassConstant(class)
}
