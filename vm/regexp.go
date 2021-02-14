package vm

import (
	"regexp"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// Regexp type alias for regexp
type Regexp = regexp.Regexp

// RegexpObject represents regexp instances, which of the type is actually string.
type RegexpObject struct {
	BaseObj
	regexp *Regexp
}

var regexpClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			arg, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, arg.Class().Name)
			}

			r := initRegexpObject(t.vm, args[0].ToString(t))
			if r == nil {
				return t.vm.InitErrorObject(t, errors.ArgumentError, "Invalid regexp: %v", args[0].ToString(t))
			}
			return r
		},
	},
}

var regexpInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "match?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			arg := args[0]
			input, ok := arg.(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, arg.Class().Name)
			}

			re := receiver.(*RegexpObject).regexp
			m := re.MatchString(string(input))

			return BooleanObject(m)
		},
		Primitive: true,
	},
}

func initRegexpObject(vm *VM, regexpStr string) *RegexpObject {
	r, err := regexp.Compile(regexpStr)
	if err != nil {
		return nil
	}
	return &RegexpObject{
		BaseObj: BaseObj{class: vm.TopLevelClass(classes.RegexpClass)},
		regexp:  r,
	}
}

func initRegexpClass(vm *VM) *RClass {
	return vm.InitClass(classes.RegexpClass).
		ClassMethods(regexpClassMethods).
		InstanceMethods(regexpInstanceMethods)
}

// Value returns the object
func (r *RegexpObject) Value() interface{} {
	return r.regexp.String()
}

// ToString returns the object's name as the string format
func (r *RegexpObject) ToString(t *Thread) string {
	return r.regexp.String()
}

// Inspect delegates to ToString
func (r *RegexpObject) Inspect(t *Thread) string {
	return r.ToString(t)
}

// ToJSON just delegates to ToString
func (r *RegexpObject) ToJSON(t *Thread) string {
	return "\"" + r.ToString(t) + "\""
}

// EqualTo returns true if the RegexpObject is equal to the other object
func (r *RegexpObject) EqualTo(with Object) bool {
	right, ok := with.(*RegexpObject)
	if !ok {
		return false
	}

	if r.Value() == right.Value() {
		return true
	}

	return false
}
