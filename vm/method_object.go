package vm

import (
	"fmt"
	"strings"

	"github.com/robotii/lito/compiler/bytecode"
	"github.com/robotii/lito/vm/classes"
)

// MethodObject represents methods defined using Lito.
type MethodObject struct {
	BaseObj
	Name           string
	instructionSet *bytecode.InstructionSet
	argc           int
}

func initMethodClass(vm *VM) *RClass {
	return vm.InitClass(classes.MethodClass)
}

// ToString returns a string representation of the method
func (m *MethodObject) ToString(t *Thread) string {
	var out strings.Builder

	out.WriteString(fmt.Sprintf("<Method: %s (%d params)\n>", m.Name, m.argc))
	out.WriteString(m.instructionSet.Inspect())

	return out.String()
}

// Inspect delegates to ToString
func (m *MethodObject) Inspect(t *Thread) string {
	return m.ToString(t)
}

// ToJSON returns the method as a JSON string
func (m *MethodObject) ToJSON(t *Thread) string {
	return m.ToString(t)
}

// Value returns method object's string format
func (m *MethodObject) Value() interface{} {
	return m.ToString(nil)
}

func (m *MethodObject) paramTypes() []uint8 {
	return m.instructionSet.ArgTypes.Types()
}

func (m *MethodObject) isSplatArgIncluded() bool {
	for _, argType := range m.paramTypes() {
		if argType == bytecode.SplatArg {
			return true
		}
	}

	return false
}

func (m *MethodObject) isKeywordArgIncluded() bool {
	for _, argType := range m.paramTypes() {
		if argType == bytecode.OptionalKeywordArg || argType == bytecode.RequiredKeywordArg {
			return true
		}
	}

	return false
}
