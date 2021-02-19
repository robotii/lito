package vm

import (
	"sync"

	"github.com/robotii/lito/compiler/bytecode"
)

type callFrameStack struct {
	callFrames []callFrame
	pointer    int
}

type baseFrame struct {
	self       Object // self points to the object used as receiver
	isBlock    bool
	isRemoved  bool // for helping stop the frame execution
	blockFrame *CallFrame
	sync.RWMutex
	sourceLine int
	fileName   string
}

type callFrame interface {
	// Getters
	Self() Object
	BlockFrame() *CallFrame
	IsBlock() bool
	IsRemoved() bool
	SourceLine() int
	FileName() string

	// Actions
	setAsRemoved()
	stopExecution()

	// Constants
	storeConstant(constName string, constant Object) *Pointer
	lookupConstantUnderAllScope(constName string) *Pointer
	lookupConstantUnderCurrentScope(constName string) *Pointer
	lookupConstantInCurrentScope(constName string) *Pointer
	inspect() string
}

type goCallFrame struct {
	baseFrame
	method   Method
	argPtr   int
	argCount int
	name     string
}

func (cf *goCallFrame) stopExecution() {}

// CallFrame structure to hold a callframe
type CallFrame struct {
	baseFrame
	locals         []*Pointer               // local variables
	ep             *CallFrame               // environment pointer, points to the call frame we want to get locals from
	instructionSet *bytecode.InstructionSet // bytecode to execute
	pc             int                      // program counter
}

func (cf *CallFrame) instructionsCount() int {
	return cf.instructionSet.Count
}

func (cf *CallFrame) stopExecution() {
	cf.pc = cf.instructionsCount()
}

// IsEmpty returns true if there are no instructions in the callframe
func (cf *CallFrame) IsEmpty() bool {
	return len(cf.instructionSet.Instructions) == 0 || cf.instructionSet.Instructions[0].Opcode == bytecode.Leave
}

func (cf *baseFrame) Self() Object {
	return cf.self
}

func (cf *baseFrame) BlockFrame() *CallFrame {
	return cf.blockFrame
}

func (cf *baseFrame) IsBlock() bool {
	return cf.isBlock
}

func (cf *baseFrame) IsRemoved() bool {
	return cf.isRemoved
}

func (cf *baseFrame) setAsRemoved() {
	cf.isRemoved = true
	if cf.isBlock && cf.blockFrame != nil {
		cf.blockFrame.isRemoved = true
	}
}

func (cf *baseFrame) SourceLine() int {
	return cf.sourceLine
}

func (cf *baseFrame) FileName() string {
	return cf.fileName
}

// TODO: see if we can remove the locking here
func (cf *CallFrame) getLocal(index, depth int) (p *Pointer) {
	lcf := cf
	for depth > 0 {
		lcf = lcf.blockFrame.ep
		depth--
	}

	lcf.RLock()
	defer lcf.RUnlock()

	if index < len(lcf.locals) {
		p = lcf.locals[index]
	}
	return
}

func (cf *CallFrame) getLocalFast(index int) (p *Pointer) {
	if index < len(cf.locals) {
		p = cf.locals[index]
	}
	return
}

func (cf *CallFrame) insertLocal(index, depth int, value Object) {
	existingLocal := cf.getLocal(index, depth)
	if existingLocal != nil {
		existingLocal.Target = value
		return
	}

	cf.Lock()
	defer cf.Unlock()

	if cf.locals == nil {
		cf.locals = make([]*Pointer, index+1)
	} else {
		for index >= len(cf.locals) {
			cf.locals = append(cf.locals, nil)
		}
	}

	cf.locals[index] = &Pointer{Target: value}
}

func (cf *CallFrame) initLocals(size int) {
	cf.locals = make([]*Pointer, size)
	for i := range cf.locals {
		cf.locals[i] = &Pointer{}
	}
}

func (cf *CallFrame) initLocalsFrom(objs ...Object) {
	cf.locals = make([]*Pointer, len(objs))
	for i, v := range objs {
		cf.locals[i] = &Pointer{Target: v}
	}
}

func (cf *CallFrame) insertLocalFast(index int, o Object) {
	if cf.locals == nil {
		cf.locals = make([]*Pointer, index+1)
	} else {
		for index >= len(cf.locals) {
			cf.locals = append(cf.locals, nil)
		}
	}

	if cf.locals[index] != nil {
		cf.locals[index].Target = o
	} else {
		cf.locals[index] = &Pointer{Target: o}
	}
}

func (cf *baseFrame) storeConstant(constName string, constant Object) (ptr *Pointer) {

	ptr = &Pointer{Target: constant}

	switch scope := cf.self.(type) {
	case *RClass:
		scope.constants[constName] = ptr
		if class, ok := ptr.Target.(*RClass); ok {
			class.scope = scope
		}
	default:
		c := cf.self.Class()
		c.constants[constName] = ptr
	}

	return
}

func (cf *baseFrame) lookupConstantUnderAllScope(constName string) *Pointer {
	var c *Pointer

	switch scope := cf.self.(type) {
	case *RClass:
		c = scope.lookupConstantUnderAllScope(constName)
	default:
		if scope == nil {
			return nil
		}
		scopeClass := scope.Class()
		c = scopeClass.lookupConstantUnderAllScope(constName)
	}

	return c
}

func (cf *baseFrame) lookupConstantUnderCurrentScope(constName string) *Pointer {
	var c *Pointer

	switch scope := cf.self.(type) {
	case *RClass:
		c = scope.lookupConstantUnderCurrentScope(constName)
	default:
		scopeClass := scope.Class()
		c = scopeClass.lookupConstantUnderCurrentScope(constName)
	}

	return c
}

func (cf *baseFrame) lookupConstantInCurrentScope(constName string) *Pointer {
	var c *Pointer

	switch scope := cf.self.(type) {
	case *RClass:
		c = scope.lookupConstantInCurrentScope(constName)
	default:
		scopeClass := scope.Class()
		c = scopeClass.lookupConstantInCurrentScope(constName)
	}

	return c
}

func (cfs *callFrameStack) push(cf callFrame) {
	if cf == nil {
		panic("Callframe can't be nil!")
	}

	if len(cfs.callFrames) <= cfs.pointer {
		cfs.callFrames = append(cfs.callFrames, cf)
	} else {
		cfs.callFrames[cfs.pointer] = cf
	}

	cfs.pointer++
}

func (cfs *callFrameStack) pop() callFrame {
	var cf callFrame

	if len(cfs.callFrames) < 1 {
		panic("Nothing to pop!")
	}

	if cfs.pointer > 0 {
		cfs.pointer--
	}

	cf = cfs.callFrames[cfs.pointer]
	if cfs.pointer > 0 {
		cfs.callFrames[cfs.pointer] = nil
	}
	return cf
}

func (cfs *callFrameStack) top() callFrame {
	if cfs.pointer > 0 {
		return cfs.callFrames[cfs.pointer-1]
	}
	return nil
}

func newNormalCallFrame(is *bytecode.InstructionSet, filename string, sourceLine int) *CallFrame {
	return &CallFrame{baseFrame: baseFrame{fileName: filename, sourceLine: sourceLine}, instructionSet: is}
}

func newGoCallFrame(m Method, receiver Object, argCount, argPtr int, n, filename string, sourceLine int, blockFrame *CallFrame) *goCallFrame {
	return &goCallFrame{
		baseFrame: baseFrame{
			self:       receiver,
			fileName:   filename,
			sourceLine: sourceLine,
			blockFrame: blockFrame,
		},
		method:   m,
		name:     n,
		argCount: argCount,
		argPtr:   argPtr,
	}
}

func reuseGoCallFrame(f *goCallFrame, m Method, receiver Object, argCount, argPtr int, n, filename string, sourceLine int, blockFrame *CallFrame) {
	f.fileName = filename
	f.sourceLine = sourceLine
	f.blockFrame = blockFrame
	f.method = m
	f.name = n
	f.self = receiver
	f.argCount = argCount
	f.argPtr = argPtr
}
