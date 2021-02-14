package vm

import (
	"github.com/robotii/lito/compiler/bytecode"
	"github.com/robotii/lito/compiler/parser"
)

const repl = "REPL"

// InitForREPL does following things:
// - Set vm to REPL mode
// - Create and push main object frame
func (vm *VM) InitForREPL() {

	// REPL should maintain a base call frame so that the whole program won't exit
	cf := newNormalCallFrame(&bytecode.InstructionSet{Name: repl}, repl, 1)
	cf.self = vm.mainObj
	vm.mode = parser.REPLMode
	vm.mainThread.callFrameStack.push(cf)
}

// REPLExec executes instructions differently from normal program execution.
func (vm *VM) REPLExec(sets []*bytecode.InstructionSet) {
	// TODO: We need to capture the current line to make the stack traces correct
	program := vm.transferProgram(repl, sets)
	oldFrame := vm.mainThread.callFrameStack.pop()
	// TODO: Find a better way to do this
	for vm.mainThread.callFrameStack.pointer > 0 {
		oldFrame = vm.mainThread.callFrameStack.pop()
	}
	// TODO: Find a better way to do this
	for vm.mainThread.Stack.pointer > 0 {
		vm.mainThread.Stack.Pop()
	}

	// Handle the case where the existing frame is gone
	// TODO: Find a better way of dealing with this
	// TODO: Not sure we need this any more? Check the pop callstack method to be sure
	if oldFrame == nil {
		f := newNormalCallFrame(&bytecode.InstructionSet{Name: repl}, repl, 1)
		f.self = vm.mainObj
		oldFrame = f
	}
	vm.mainThread.callFrameStack.pointer = 1
	cf := newNormalCallFrame(program, repl, oldFrame.SourceLine())
	if ocf, ok := oldFrame.(*CallFrame); ok {
		cf.locals = ocf.locals
		cf.ep = ocf.ep
	}

	cf.isBlock = oldFrame.IsBlock()
	cf.self = oldFrame.Self()

	defer func() {
		e := recover()

		switch err := e.(type) {
		case error:
			panic(err)
		}
	}()

	// Now evaluate the frame
	vm.mainThread.evaluateNormalFrame(cf)
	// Copy the locals back to the oldframe
	if ocf, ok := oldFrame.(*CallFrame); ok {
		ocf.locals = cf.locals
	}
	// Push callframe onto the stack, as it was popped off
	// This will be reused for the next call to REPLExec.
	vm.mainThread.callFrameStack.push(cf)
}

// GetExecResult returns stack's top most value. Normally it's used in tests.
func (vm *VM) GetExecResult() Object {
	top := vm.mainThread.Stack.top()
	if top != nil {
		return top
	}
	return NIL
}

// GetBaseFrame returns the bottom frame for use by the REPL
func (vm *VM) GetBaseFrame() *CallFrame {
	cf := vm.mainThread.callFrameStack.callFrames[0]
	if callFrame, ok := cf.(*CallFrame); ok {
		return callFrame
	}
	return nil
}

// GetREPLResult returns a string that should be shown after each evaluation.
func (vm *VM) GetREPLResult() string {
	// Avoid nasty error on stack if we execute an instruction that
	// leaves nothing on the stack, such as `break`
	if len(vm.mainThread.Stack.data) > 0 {
		top := vm.mainThread.Stack.Pop()
		if top != nil {
			return top.ToString(&vm.mainThread)
		}
	}
	return ""
}
