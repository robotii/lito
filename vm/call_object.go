package vm

import (
	"fmt"

	"github.com/robotii/lito/compiler/bytecode"
)

type callObject struct {
	method       *MethodObject
	receiverPtr  int
	argCount     int
	argSet       *bytecode.ArgSet
	argIndex     int
	lastArgIndex int
	callFrame    *CallFrame
}

func (co *callObject) paramTypes() []uint8 {
	return co.method.instructionSet.ArgTypes.Types()
}

func (co *callObject) paramNames() []string {
	return co.method.instructionSet.ArgTypes.Names()
}

func (co *callObject) methodName() string {
	return co.method.Name
}

func (co *callObject) argTypes() []uint8 {
	if co.argSet == nil {
		return []uint8{}
	}

	return co.argSet.Types()
}

func (co *callObject) argPtr() int {
	return co.receiverPtr + 1
}

func (co *callObject) argPosition() int {
	return co.argPtr() + co.argIndex
}

func (co *callObject) assignNormalArguments(stack []Object) {
	for i, paramType := range co.paramTypes() {
		if paramType == bytecode.NormalArg {
			co.callFrame.insertLocalFast(i, stack[co.argPosition()])
			co.argIndex++
		}
	}
}

func (co *callObject) assignNormalAndOptionedArguments(paramIndex int, stack []Object) {
	for argIndex, at := range co.argTypes() {
		if co.lastArgIndex < argIndex && (at == bytecode.NormalArg || at == bytecode.OptionedArg) {
			co.callFrame.insertLocalFast(paramIndex, stack[co.argPtr()+argIndex])
			co.lastArgIndex = argIndex
			break
		}
	}
}

func (co *callObject) assignKeywordArguments(stack []Object) (err error) {
	for argIndex, argType := range co.argTypes() {
		if argType == bytecode.RequiredKeywordArg || argType == bytecode.OptionalKeywordArg {
			argName := co.argSet.Names()[argIndex]
			paramIndex, ok := co.hasKeywordParam(argName)
			if ok {
				co.callFrame.insertLocalFast(paramIndex, stack[co.argPtr()+argIndex])
			} else {
				err = fmt.Errorf("unknown key %s for method %s", argName, co.methodName())
			}
		}
	}
	return
}

func (co *callObject) assignSplatArgument(stack []Object, arr *ArrayObject) {
	index := len(co.paramTypes()) - 1

	for co.argIndex < co.argCount {
		arr.Elements = append(arr.Elements, stack[co.argPosition()])
		co.argIndex++
	}

	co.callFrame.insertLocalFast(index, arr)
}

func (co *callObject) hasKeywordParam(name string) (index int, result bool) {
	for paramIndex, paramType := range co.paramTypes() {
		paramName := co.paramNames()[paramIndex]
		if paramName == name && (paramType == bytecode.RequiredKeywordArg || paramType == bytecode.OptionalKeywordArg) {
			index = paramIndex
			result = true
			return
		}
	}
	return
}

func (co *callObject) hasKeywordArgument(name string) (index int, result bool) {
	for argIndex, argType := range co.argTypes() {
		argName := co.argSet.Names()[argIndex]
		if argName == name && (argType == bytecode.RequiredKeywordArg || argType == bytecode.OptionalKeywordArg) {
			index = argIndex
			result = true
			return
		}
	}
	return
}

func (co *callObject) normalParamsCount() (n int) {
	for _, at := range co.paramTypes() {
		if at == bytecode.NormalArg {
			n++
		}
	}
	return
}

func (co *callObject) normalArgsCount() (n int) {
	for _, at := range co.argTypes() {
		if at == bytecode.NormalArg {
			n++
		}
	}
	return
}
