package vm

import (
	"io/ioutil"
	"path/filepath"

	"github.com/robotii/lito/compiler"
	"github.com/robotii/lito/compiler/bytecode"
	"github.com/robotii/lito/compiler/parser"
	"github.com/robotii/lito/vm/errors"
)

const mainThreadID = 0

// Thread is the context needed for a single thread of execution
type Thread struct {
	// a stack that holds call frames
	callFrameStack callFrameStack
	// The call frame currently being executed
	currentFrame callFrame
	// cachedFrame stores a per thread call frame for reuse
	cachedFrame goCallFrame
	// the current line being executed
	currentLine int
	// data Stack
	Stack Stack
	// theads have an id so they can be looked up in the vm. The main thread is always 0
	id int64
	vm *VM
}

// VM returns the VM associated with the thread
func (t *Thread) VM() *VM {
	return t.vm
}

func (t *Thread) isMainThread() bool {
	return t.id == mainThreadID
}

// GetSourceLine returns the current source line
func (t *Thread) GetSourceLine() int {
	return t.currentLine
}

func (t *Thread) loadLibrary(libName string) (err error) {
	libPath := filepath.Join(t.vm.libPath, libName)
	err = t.execFile(libPath)
	return
}

func (t *Thread) execFile(fpath string) (err error) {
	file, err := ioutil.ReadFile(fpath)
	if err != nil {
		return
	}

	instructionSets, err := compiler.CompileToInstructions(string(file), parser.NormalMode)
	if err != nil {
		return
	}

	t.vm.ExecInstructions(instructionSets, fpath)
	return
}

func (t *Thread) evaluateGoFrame(cf *goCallFrame) {
	// Error handling
	defer func() {
		if r := recover(); r != nil {
			t.reportErrorAndStop(r)
		}
	}()
	// Push the frame onto the stack
	t.callFrameStack.push(cf)
	// Save the current frame to restore afterwards
	oldFrame := t.currentFrame
	t.currentFrame = cf
	args := t.Stack.data[cf.argPtr : cf.argPtr+cf.argCount]
	result := cf.method(cf.self, t, args)
	t.Stack.Push(result)
	if !cf.IsRemoved() {
		t.callFrameStack.pop()
	}
	// restore the current frame
	t.currentFrame = oldFrame
}

func (t *Thread) evaluateNormalFrame(cf *CallFrame) {
	t.callFrameStack.push(cf)
	defer func() {
		if r := recover(); r != nil {
			t.reportErrorAndStop(r)
		}
	}()
	// Save the current frame to restore afterwards
	oldFrame := t.currentFrame
	t.currentFrame = cf

	t.execFrame(cf)
	if !cf.IsRemoved() {
		cf.isRemoved = true
		t.callFrameStack.pop()
	}
	// restore the current frame
	t.currentFrame = oldFrame
}

func (t *Thread) reportErrorAndStop(e interface{}) {
	top := t.Stack.top()
	switch err := top.(type) {
	// If we can get an error object it means it's a Lito error
	case *Error:
		if !err.storedTraces {
			err.storeStackTraces(t)
		}
		t.stopFrame()
		panic(err)
		// Otherwise it's a Go panic that needs to be raised
	default:
		t.stopFrame()
		panic(e)
	}
}

func (t *Thread) stopFrame() {
	if cf := t.callFrameStack.top(); cf != nil {
		cf.stopExecution()
		if !cf.IsRemoved() {
			t.callFrameStack.pop()
		}
	}
}

// BlockGiven returns whether or not we have a block frame below us in the stack
func (t *Thread) BlockGiven() bool {
	return t.GetBlock() != nil
}

// GetBlock returns the current block
func (t *Thread) GetBlock() *CallFrame {
	return t.currentFrame.BlockFrame()
}

// Yield executes a block frame and returns the result
func (t *Thread) Yield(blockFrame *CallFrame, args ...Object) Object {
	return t.YieldWithBlockArgument(blockFrame, blockFrame, args...)
}

// YieldWithBlockArgument executes a block frame and returns the result
func (t *Thread) YieldWithBlockArgument(blockFrame *CallFrame, block *CallFrame, args ...Object) Object {
	if blockFrame.IsRemoved() {
		return NIL
	}

	c := newNormalCallFrame(blockFrame.instructionSet, blockFrame.FileName(), blockFrame.sourceLine)
	c.blockFrame = block
	c.ep = blockFrame.ep
	c.self = blockFrame.self
	c.isBlock = true
	c.initLocalsFrom(args...)

	t.evaluateNormalFrame(c)

	if blockFrame.IsRemoved() {
		return NIL
	}

	return t.Stack.top()
}

func (t *Thread) sendMethod(methodName string, argCount int, blockFrame *CallFrame) {
	// Splat the current block if it is the last argument
	// Check if we have an argument, as we don't want to splat the receiver
	if blk, ok := t.Stack.top().(*BlockObject); ok && argCount > 0 && blk.splat {
		// Pop block
		t.Stack.Discard()
		blockFrame = blk.asCallFrame(t)
		argCount--
	}
	argCount = t.unsplatArray(argCount)

	argPr := t.Stack.pointer - argCount - 1
	receiverPr := argPr - 1
	receiver := t.Stack.data[receiverPr]

	// TODO: Use copy here
	for i := 0; i < argCount; i++ {
		t.Stack.data[argPr+i] = t.Stack.data[argPr+i+1]
	}

	t.Stack.pointer--

	t.FindAndExecute(receiver, methodName, false, receiverPr, argPr, argCount, nil, blockFrame, t.callFrameStack.top().FileName())
}

func (t *Thread) evalBuiltinMethod(receiver Object, method *BuiltinMethodObject, receiverPtr, argCount int, argSet *bytecode.ArgSet, blockFrame *CallFrame, fileName string) {
	var cf *goCallFrame
	argPtr := receiverPtr + 1
	sourceLine := t.GetSourceLine()

	if !method.Primitive {
		cf = newGoCallFrame(
			method.Fn, receiver, argCount, argPtr,
			method.Name, fileName, sourceLine, blockFrame,
		)
	} else {
		reuseGoCallFrame(&t.cachedFrame,
			method.Fn, receiver, argCount, argPtr,
			method.Name, fileName, sourceLine, blockFrame,
		)
		cf = &t.cachedFrame
	}

	t.evaluateGoFrame(cf)
	evaluated := t.Stack.top()

	// Special case the new method to call init
	_, ok := receiver.(*RClass)
	if ok && method.Name == "new" {
		if instance, ok := evaluated.(*RObject); ok {
			init, exists := instance.class.lookupMethod(initMethod).(*MethodObject)
			if exists && init != nil {
				t.evalMethodCall(instance, init, receiverPtr, argCount, argSet, blockFrame, sourceLine)
			}
		}
	}

	t.Stack.Set(receiverPtr, evaluated)
	t.Stack.pointer = cf.argPtr

	// Check for an error that has been raised
	if err, ok := evaluated.(*Error); ok {
		if err.Ignore && err.Raised {
			err.Ignore = false
		} else if err.Raised {
			panic(err)
		}
	}
}

func (t *Thread) evalMethodCall(receiver Object, method *MethodObject, receiverPtr, argCount int, argSet *bytecode.ArgSet, blockFrame *CallFrame, sourceLine int) {
	cf := newNormalCallFrame(method.instructionSet, method.instructionSet.Filename, sourceLine)
	cf.self = receiver
	cf.blockFrame = blockFrame

	call := callObject{
		method:      method,
		receiverPtr: receiverPtr,
		argCount:    argCount,
		argSet:      argSet,
		// This is only for normal/optioned arguments
		lastArgIndex: -1,
		callFrame:    cf,
	}
	normalParamsCount := call.normalParamsCount()
	paramTypes := call.paramTypes()
	paramsCount := len(call.paramTypes())
	stack := t.Stack.data

	if call.argCount > paramsCount && !call.method.isSplatArgIncluded() {
		t.reportArgumentError(paramsCount, call.methodName(), call.argCount, call.receiverPtr)
	}

	if normalParamsCount > call.argCount {
		t.reportArgumentError(normalParamsCount, call.methodName(), call.argCount, call.receiverPtr)
	}

	// Check if arguments include all the required keys before assign keyword arguments
	for paramIndex, paramType := range paramTypes {
		switch paramType {
		case bytecode.RequiredKeywordArg:
			paramName := call.paramNames()[paramIndex]
			if _, ok := call.hasKeywordArgument(paramName); !ok {
				t.setErrorObject(call.receiverPtr, call.argPtr(), errors.ArgumentError, "Method %s requires key argument %s", call.methodName(), paramName)
			}
		}
	}

	// initialise the Locals
	cf.initLocals(paramsCount)

	// Assign the keyword args
	err := call.assignKeywordArguments(stack)
	if err != nil {
		t.setErrorObject(call.receiverPtr, call.argPtr(), errors.ArgumentError, err.Error())
	}

	// If given arguments is more than the normal arguments.
	// It might mean we have optioned argument been override.
	// Or we have some keyword arguments
	if normalParamsCount < call.argCount {
		for paramIndex, paramType := range paramTypes {
			switch paramType {
			case bytecode.NormalArg, bytecode.OptionedArg:
				call.assignNormalAndOptionedArguments(paramIndex, stack)
			case bytecode.SplatArg:
				call.argIndex = paramIndex
				call.assignSplatArgument(stack, InitArrayObject([]Object{}))
			}
		}
	} else {
		call.assignNormalArguments(stack)
	}

	t.evaluateNormalFrame(call.callFrame)

	// Put the return value on the stack
	t.Stack.Set(call.receiverPtr, t.Stack.top())
	t.Stack.pointer = call.argPtr()
}

func (t *Thread) reportArgumentError(idealArgNumber int, methodName string, exactArgNumber int, receiverPtr int) {
	var message string

	if idealArgNumber > exactArgNumber {
		message = "Expect at least %d args for method '%s'. got: %d"
	} else {
		message = "Expect at most %d args for method '%s'. got: %d"
	}

	t.setErrorObject(receiverPtr, receiverPtr+1, errors.ArgumentError, message, idealArgNumber, methodName, exactArgNumber)
}

func (t *Thread) pushErrorObject(errorType string, format string, args ...interface{}) {
	err := t.vm.InitErrorObject(t, errorType, format, args...)
	err.storeStackTraces(t)
	t.Stack.Push(err)
	panic(err)
}

func (t *Thread) setErrorObject(receiverPtr, sp int, errorType string, format string, args ...interface{}) {
	err := t.vm.InitErrorObject(t, errorType, format, args...)
	t.Stack.Set(receiverPtr, err)
	t.Stack.pointer = sp
	panic(err)
}

// FindAndExecute finds and executes a method
func (t *Thread) FindAndExecute(receiver Object, methodName string, super bool, receiverPr int, argPr int, argCount int, argSet *bytecode.ArgSet, blockFrame *CallFrame, fileName string) {
	method := receiver.FindMethod(methodName, super)

	if method == nil {
		mm := receiver.FindLookup(receiver.Class().inheritsLookup)

		if mm == nil {
			t.setErrorObject(receiverPr, argPr, errors.NoMethodError, errors.UndefinedMethod, methodName, receiver.ToString(t))
		} else {
			// Move up args for missed method's name
			t.Stack.Push(nil)
			copy(t.Stack.data[argPr+1:argPr+argCount+1], t.Stack.data[argPr:argPr+argCount])
			t.Stack.Set(argPr, StringObject(methodName))
			argCount++

			method = mm
		}
	}

	switch m := method.(type) {
	case *MethodObject:
		t.evalMethodCall(receiver, m, receiverPr, argCount, argSet, blockFrame, t.GetSourceLine())
	case *BuiltinMethodObject:
		t.evalBuiltinMethod(receiver, m, receiverPr, argCount, argSet, blockFrame, fileName)
	case *Error:
		t.pushErrorObject(errors.InternalError, m.ToString(t))
	}
}
