package bytecode

import (
	"fmt"
	"strings"
)

// instruction set types
const (
	Method  = "Method"
	Class   = "Class"
	Block   = "Block"
	Program = "Program"
)

// instruction actions
const (
	NoOp int = iota
	GetLocal
	GetConstant
	GetInstanceVariable
	SetLocal
	SetOptional
	SetConstant
	SetInstanceVariable
	PutTrue
	PutFalse
	PutString
	PutFloat
	PutSelf
	PutSuper
	PutInt
	PutObject
	PutNull
	NewArray
	ExpandArray
	SplatArray
	SplatBlock
	NewHash
	NewRange
	NewRangeExcl
	BranchUnless
	BranchIf
	Jump
	Break
	DefMethod
	DefMetaMethod
	DefClass
	Send
	BinaryOperator
	Add
	Subtract
	Greater
	Less
	GreaterEqual
	LessEqual
	InvokeBlock
	GetBlock
	HasBlock
	Pop
	Dup
	Defer
	Leave
	InstructionCount
)

// InstructionNameTable is the table the maps instruction's op code with its readable name
var InstructionNameTable = [...]string{
	NoOp:                "no_op",
	GetLocal:            "getlocal",
	GetConstant:         "getconstant",
	GetInstanceVariable: "getinstancevariable",
	SetLocal:            "setlocal",
	SetOptional:         "setoptional",
	SetConstant:         "setconstant",
	SetInstanceVariable: "setinstancevariable",
	PutTrue:             "puttrue",
	PutFalse:            "putfalse",
	PutString:           "putstring",
	PutFloat:            "putfloat",
	PutSelf:             "putself",
	PutSuper:            "putsuper",
	PutInt:              "putint",
	PutObject:           "putobject",
	PutNull:             "putnil",
	NewArray:            "newarray",
	ExpandArray:         "expand_array",
	SplatArray:          "splat_array",
	SplatBlock:          "splat_block",
	NewHash:             "newhash",
	NewRange:            "newrange",
	NewRangeExcl:        "newrangeexcl",
	BranchUnless:        "branchunless",
	BranchIf:            "branchif",
	Jump:                "jump",
	Break:               "break",
	DefMethod:           "def_method",
	DefMetaMethod:       "def_meta_method",
	DefClass:            "def_class",
	Send:                "send",
	BinaryOperator:      "bin_op",
	Add:                 "add",
	Subtract:            "subtract",
	Greater:             "greater",
	Less:                "less",
	GreaterEqual:        "greater_equal",
	LessEqual:           "less_equal",
	InvokeBlock:         "invokeblock",
	GetBlock:            "getblock",
	HasBlock:            "hasblock",
	Pop:                 "pop",
	Dup:                 "dup",
	Defer:               "defer",
	Leave:               "leave",
	InstructionCount:    "instruction_count",
}

// Instruction represents compiled bytecode instruction
type Instruction struct {
	Opcode int
	Params []interface{}
}

type anchorReference struct {
	anchor   *anchor
	insSet   *InstructionSet
	insIndex int
}
type anchor struct {
	line int
}

// Inspect is for inspecting the instruction's content
func (i *Instruction) Inspect() string {
	var params []string

	for _, param := range i.Params {
		params = append(params, fmt.Sprint(param))
	}
	return fmt.Sprintf("%s: %s.", i.ActionName(), strings.Join(params, ", "))
}

// ActionName returns the human readable name of the instruction
func (i *Instruction) ActionName() string {
	return InstructionNameTable[i.Opcode]
}

func (a *anchor) define(l int) {
	a.line = l
}

// InstructionSet contains a set of Instructions and some metadata
type InstructionSet struct {
	Name         string
	Filename     string
	Type         string
	Instructions []Instruction
	SourceMap    []int
	Count        int
	ArgTypes     ArgSet
}

// ArgSet stores the metadata of a method definition's parameters.
type ArgSet struct {
	names []string
	types []uint8
}

// Types are the getter method of *ArgSet's types attribute
// TODO: needs to change the func to simple public variable
func (as *ArgSet) Types() []uint8 {
	return as.types
}

// Names are the getter method of *ArgSet's names attribute
// TODO: needs to change the func to simple public variable
func (as *ArgSet) Names() []string {
	return as.names
}

// FindIndex finds the index of the given name in the argset
func (as *ArgSet) FindIndex(name string) int {
	for i, n := range as.names {
		if n == name {
			return i
		}
	}

	return -1
}

func (as *ArgSet) setArg(index int, name string, argType uint8) {
	as.names[index] = name
	as.types[index] = argType
}

// Inspect returns a string representation of the InstructionSet
func (is *InstructionSet) Inspect() string {
	var out strings.Builder

	for i, ins := range is.Instructions {
		out.WriteString(fmt.Sprintf("%v : %v source line: %d\n", i, ins.Inspect(), is.SourceMap[i]))
	}

	return out.String()
}
func (is *InstructionSet) define(action int, sourceLine int, params ...interface{}) *anchorReference {
	var ref *anchorReference
	is.Instructions = append(is.Instructions, Instruction{Opcode: action, Params: params})
	is.SourceMap = append(is.SourceMap, sourceLine+1)

	for _, param := range params {
		a, ok := param.(*anchor)
		if ok {
			ref = &anchorReference{anchor: a, insSet: is, insIndex: is.Count}
			break
		}
	}

	is.Count++
	return ref
}

func (is *InstructionSet) elide() {
	for i, v := range is.Instructions {
		switch v.Opcode {
		case Jump:
			if target, ok := v.Params[0].(int); ok {
				if target >= len(is.Instructions) || is.Instructions[target].Opcode == Leave {
					v.Opcode = Leave
					is.Instructions[i] = v
					continue
				}
				if is.Instructions[target].Opcode == Jump {
					v.Params[0] = is.Instructions[target].Params[0]
					is.Instructions[i] = v
				}
			}
		}
	}
}
