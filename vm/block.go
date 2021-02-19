package vm

import (
	"fmt"

	"github.com/robotii/lito/compiler/bytecode"
	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// BlockObject represents an instance of `Block` class.
type BlockObject struct {
	BaseObj
	instructionSet *bytecode.InstructionSet
	ep             *CallFrame
	self           Object
	splat          bool
}

var blockClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.ArgumentError, "Can't create block object without block argument")
			}
			return initBlockObject(t.vm, blockFrame.instructionSet, blockFrame.ep, blockFrame.self)
		},
	},
}

var blockInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "call",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			block := receiver.(*BlockObject)
			c := block.asCallFrame(t)
			// Deal with a block being passed
			blockFrame := t.GetBlock()
			if blockFrame != nil {
				return t.YieldWithBlockArgument(c, blockFrame, args...)
			}
			return t.Yield(c, args...)
		},
	},
}

func initBlockClass(vm *VM) *RClass {
	return vm.InitClass(classes.BlockClass).
		ClassMethods(blockClassMethods).
		InstanceMethods(blockInstanceMethods)
}

func initBlockObject(vm *VM, is *bytecode.InstructionSet, ep *CallFrame, self Object) *BlockObject {
	return &BlockObject{
		BaseObj:        BaseObj{class: vm.TopLevelClass(classes.BlockClass)},
		instructionSet: is,
		ep:             ep,
		self:           self,
	}
}

func (bo *BlockObject) asCallFrame(t *Thread) *CallFrame {
	c := newNormalCallFrame(bo.instructionSet, bo.instructionSet.Filename, bo.instructionSet.SourceMap[0])
	c.ep = bo.ep
	c.self = bo.self
	c.isBlock = true
	return c
}

// Value returns the object
func (bo *BlockObject) Value() interface{} {
	return bo.instructionSet
}

// ToString returns the object's name as the string format
func (bo *BlockObject) ToString(t *Thread) string {
	return fmt.Sprintf("<Block: %s>", bo.instructionSet.Filename)
}

// Inspect delegates to ToString
func (bo *BlockObject) Inspect(t *Thread) string {
	return bo.ToString(t)
}

// ToJSON just delegates to ToString
func (bo *BlockObject) ToJSON(t *Thread) string {
	return bo.ToString(t)
}

// copy returns the duplicate of the Block object
func (bo *BlockObject) copy() Object {
	return &BlockObject{
		BaseObj:        BaseObj{class: bo.Class()},
		instructionSet: bo.instructionSet,
		ep:             bo.ep,
		self:           bo.self,
	}
}
