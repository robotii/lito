package vm

import (
	"fmt"

	"github.com/robotii/lito/vm/classes"
)

// GoObject is used to hold opaque Golang values that cannot be converted to Objects
type GoObject struct {
	BaseObj
	data interface{}
}

var goObjectClassMethods = []*BuiltinMethodObject{}

var goObjectInstanceMethods = []*BuiltinMethodObject{}

func initGoObject(vm *VM, d interface{}) *GoObject {
	return &GoObject{data: d, BaseObj: BaseObj{class: vm.TopLevelClass(classes.GoObjectClass)}}
}

func initGoClass(vm *VM) *RClass {
	return vm.InitClass(classes.GoObjectClass).
		ClassMethods(goObjectClassMethods).
		InstanceMethods(goObjectInstanceMethods)
}

// Value returns the object
func (s *GoObject) Value() interface{} {
	return s.data
}

// ToString returns the object's name as the string format
func (s *GoObject) ToString(t *Thread) string {
	return fmt.Sprintf("<GoObject: %p>", s)
}

// Inspect delegates to ToString
func (s *GoObject) Inspect(t *Thread) string {
	return s.ToString(t)
}

// ToJSON just delegates to ToString
func (s *GoObject) ToJSON(t *Thread) string {
	return s.ToString(t)
}
