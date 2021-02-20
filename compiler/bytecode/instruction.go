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

// NoSuperClass represents a class without a superclass.
// Using this will result in the superclass being Object.
const (
	NoSuperClass = "__none__"
)

// instruction actions
const (
	NoOp int = iota
	GetLocal
	GetConstant
	GetConstantNamespace
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
	NoOp:                 "no_op",
	GetLocal:             "getlocal",
	GetConstant:          "getconstant",
	GetConstantNamespace: "getconstantnamespace",
	GetInstanceVariable:  "getinstancevariable",
	SetLocal:             "setlocal",
	SetOptional:          "setoptional",
	SetConstant:          "setconstant",
	SetInstanceVariable:  "setinstancevariable",
	PutTrue:              "puttrue",
	PutFalse:             "putfalse",
	PutString:            "putstring",
	PutFloat:             "putfloat",
	PutSelf:              "putself",
	PutSuper:             "putsuper",
	PutInt:               "putint",
	PutObject:            "putobject",
	PutNull:              "putnil",
	NewArray:             "newarray",
	ExpandArray:          "expand_array",
	SplatArray:           "splat_array",
	SplatBlock:           "splat_block",
	NewHash:              "newhash",
	NewRange:             "newrange",
	NewRangeExcl:         "newrangeexcl",
	BranchUnless:         "branchunless",
	BranchIf:             "branchif",
	Jump:                 "jump",
	Break:                "break",
	DefMethod:            "def_method",
	DefMetaMethod:        "def_meta_method",
	DefClass:             "def_class",
	Send:                 "send",
	BinaryOperator:       "bin_op",
	Add:                  "add",
	Subtract:             "subtract",
	Greater:              "greater",
	Less:                 "less",
	GreaterEqual:         "greater_equal",
	LessEqual:            "less_equal",
	InvokeBlock:          "invokeblock",
	GetBlock:             "getblock",
	HasBlock:             "hasblock",
	Pop:                  "pop",
	Dup:                  "dup",
	Defer:                "defer",
	Leave:                "leave",
	InstructionCount:     "instruction_count",
}

type anchorReference struct {
	anchor   *anchor
	insSet   *InstructionSet
	insIndex int
}
type anchor struct {
	line int
}

func (a *anchor) define(l int) {
	a.line = l
}

// InstructionSet contains a set of Instructions and some metadata
type InstructionSet struct {
	Name         string
	Filename     string
	Type         string
	Instructions []int
	Constants    []interface{}
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
		out.WriteString(fmt.Sprintf("%v : %v source line: %d\n", i, InstructionNameTable[ins], is.SourceMap[i]))
	}

	return out.String()
}
func (is *InstructionSet) define(action int, sourceLine int, params ...interface{}) *anchorReference {
	var ref *anchorReference
	is.Instructions = append(is.Instructions, action)
	is.SourceMap = append(is.SourceMap, sourceLine+1)
	is.Count++
	for _, param := range params {
		if i, ok := param.(int); ok {
			is.Instructions = append(is.Instructions, i)
			is.SourceMap = append(is.SourceMap, sourceLine+1)
		} else {
			ci := is.SetConstant(param)
			is.Instructions = append(is.Instructions, ci)
			is.SourceMap = append(is.SourceMap, sourceLine+1)
			a, ok := param.(*anchor)
			if ok {
				ref = &anchorReference{anchor: a, insSet: is, insIndex: is.Count}
			}
		}
		is.Count++
	}
	return ref
}

// GetString returns the value of the string from the constant table
func (is *InstructionSet) GetString(index int) string {
	s, _ := is.Constants[is.Instructions[index]].(string)
	return s
}

// GetBool returns a bool value from the constant pool
func (is *InstructionSet) GetBool(index int) bool {
	b, _ := is.Constants[is.Instructions[index]].(bool)
	return b
}

// GetObject returns an interface{} object
func (is *InstructionSet) GetObject(index int) interface{} {
	return is.Constants[is.Instructions[index]]
}

// GetFloat returns the int value from the constant pool
func (is *InstructionSet) GetFloat(index int) float64 {
	f, _ := is.Constants[is.Instructions[index]].(float64)
	return f
}

// SetConstant adds a constant to the constant pool
func (is *InstructionSet) SetConstant(o interface{}) int {
	for i, v := range is.Constants {
		if o == v {
			return i
		}
	}
	is.Constants = append(is.Constants, o)
	return len(is.Constants) - 1
}
