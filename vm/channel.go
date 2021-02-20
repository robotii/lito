package vm

import (
	"fmt"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// ChannelObject represents a Golang channel.
type ChannelObject struct {
	BaseObj
	Chan         chan *Object
	ChannelState int
}

// Channel state
const (
	chOpen = iota
	chClosed
)

var channelClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) == 1 {
				arg, ok := args[0].(IntegerObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
				}
				if arg < 0 {
					return t.vm.InitErrorObject(t, errors.ArgumentError, errors.NegativeValue, int(arg))
				}
				c := &ChannelObject{BaseObj: BaseObj{class: t.vm.TopLevelClass(classes.ChannelClass)}, Chan: make(chan *Object, arg)}
				return c
			}
			c := &ChannelObject{BaseObj: BaseObj{class: t.vm.TopLevelClass(classes.ChannelClass)}, Chan: make(chan *Object, chOpen)}
			return c
		},
		Primitive: true,
	},
}

var channelInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "close",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			c := receiver.(*ChannelObject)
			if c.isClosed() {
				return t.vm.InitErrorObject(t, errors.ChannelCloseError, errors.ChannelIsClosed)
			}
			c.ChannelState = chClosed
			close(c.Chan)
			return NIL
		},
		Primitive: true,
	},
	{
		Name: "<-",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) < 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentMore, 1, len(args))
			}

			c := receiver.(*ChannelObject)
			if c.isClosed() {
				return t.vm.InitErrorObject(t, errors.ChannelCloseError, errors.ChannelIsClosed)
			}
			// Make a new variable to make the pointer work
			for _, o := range args {
				obj := o
				c.Chan <- &obj
			}
			return c
		},
		Primitive: true,
	},
	{
		Name: "receive",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return receiver.(*ChannelObject).receive(t)
		},
		Primitive: true,
	},
	{
		Name: "each",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			c := receiver.(*ChannelObject)
			if c.isClosed() {
				return t.vm.InitErrorObject(t, errors.ChannelCloseError, errors.ChannelIsClosed)
			}

			for val := range c.Chan {
				t.Yield(blockFrame, *val)
				if blockFrame.IsRemoved() {
					break
				}
			}

			return receiver
		},
	},
	{
		Name: "cap",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return IntegerObject(cap(receiver.(*ChannelObject).Chan))
		},
		Primitive: true,
	},
}

func initChannelClass(vm *VM) *RClass {
	return vm.InitClass(classes.ChannelClass).
		ClassMethods(channelClassMethods).
		InstanceMethods(channelInstanceMethods)
}

func (co *ChannelObject) isClosed() bool {
	return co.ChannelState == chClosed
}

func (co *ChannelObject) receive(t *Thread) Object {
	if co.isClosed() {
		return t.vm.InitErrorObject(t, errors.ChannelCloseError, errors.ChannelIsClosed)
	}

	obj := <-co.Chan
	if obj == nil {
		return NIL
	}
	return *obj
}

// Value returns the object
func (co *ChannelObject) Value() interface{} {
	return co.Chan
}

// ToString returns the object's name as the string format
func (co *ChannelObject) ToString(t *Thread) string {
	return fmt.Sprintf("<Channel: %p>", co.Chan)
}

// Inspect delegates to ToString
func (co *ChannelObject) Inspect(t *Thread) string {
	return co.ToString(t)
}

// ToJSON just delegates to ToString
func (co *ChannelObject) ToJSON(t *Thread) string {
	return co.ToString(t)
}

// copy returns the duplicate of the ChannelObject
func (co *ChannelObject) copy() Object {
	newC := &ChannelObject{BaseObj: BaseObj{class: co.class}, Chan: make(chan *Object, cap(co.Chan))}
	return newC
}
