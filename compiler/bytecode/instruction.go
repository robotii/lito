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

// Instructions is the table that maps instruction's opcode with its readable name
var Instructions = [...]instruction{
	NoOp:                 {"no_op", 0, nil},
	GetLocal:             {"getlocal", 2, []bool{false, false}},
	GetConstant:          {"getconstant", 1, []bool{true}},
	GetConstantNamespace: {"getconstantnamespace", 1, []bool{true}},
	GetInstanceVariable:  {"getinstancevariable", 1, []bool{true}},
	SetLocal:             {"setlocal", 2, []bool{false, false}},
	SetOptional:          {"setoptional", 2, []bool{false, false}},
	SetConstant:          {"setconstant", 1, []bool{true}},
	SetInstanceVariable:  {"setinstancevariable", 1, []bool{true}},
	PutTrue:              {"puttrue", 0, nil},
	PutFalse:             {"putfalse", 0, nil},
	PutString:            {"putstring", 1, []bool{true}},
	PutFloat:             {"putfloat", 1, []bool{true}},
	PutSelf:              {"putself", 0, nil},
	PutSuper:             {"putsuper", 0, nil},
	PutInt:               {"putint", 1, []bool{false}},
	PutObject:            {"putobject", 1, []bool{true}},
	PutNull:              {"putnil", 0, nil},
	NewArray:             {"newarray", 1, []bool{false}},
	ExpandArray:          {"expand_array", 1, []bool{false}},
	SplatArray:           {"splat_array", 0, nil},
	SplatBlock:           {"splat_block", 0, nil},
	NewHash:              {"newhash", 1, []bool{false}},
	NewRange:             {"newrange", 0, nil},
	NewRangeExcl:         {"newrangeexcl", 0, nil},
	BranchUnless:         {"branchunless", 1, []bool{false}},
	BranchIf:             {"branchif", 1, []bool{false}},
	Jump:                 {"jump", 1, []bool{false}},
	Break:                {"break", 0, nil},
	DefMethod:            {"def_method", 3, []bool{false, true, true}},
	DefMetaMethod:        {"def_meta_method", 3, []bool{false, true, true}},
	DefClass:             {"def_class", 4, []bool{true, true, true, true}},
	Send:                 {"send", 4, []bool{true, false, true, true}},
	BinaryOperator:       {"bin_op", 1, []bool{true}},
	Add:                  {"add", 1, []bool{true}},
	Subtract:             {"subtract", 1, []bool{true}},
	Greater:              {"greater", 1, []bool{true}},
	Less:                 {"less", 1, []bool{true}},
	GreaterEqual:         {"greater_equal", 1, []bool{true}},
	LessEqual:            {"less_equal", 1, []bool{true}},
	InvokeBlock:          {"invokeblock", 1, []bool{false}},
	GetBlock:             {"getblock", 0, nil},
	HasBlock:             {"hasblock", 0, nil},
	Pop:                  {"pop", 0, nil},
	Dup:                  {"dup", 0, nil},
	Defer:                {"defer", 2, []bool{true, true}},
	Leave:                {"leave", 0, nil},
	InstructionCount:     {"instruction_count", 0, nil},
}

type instruction struct {
	name       string
	paramCount int
	objects    []bool
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

	out.WriteString("Constants\n---------\n")
	for i, constant := range is.Constants {
		out.WriteString(fmt.Sprintf("%v: %v\n", i, constant))
	}

	ins := is.Instructions
	out.WriteString(fmt.Sprintf("\nBytecode: Length %v\n--------\n", len(ins)))
	for i := 0; i < len(ins); {
		opcode := ins[i]
		out.WriteString(fmt.Sprintf("%v : %v [", i, Instructions[opcode].name))
		for j := 1; j <= Instructions[opcode].paramCount; j++ {
			var paramValue string
			if Instructions[opcode].objects[j-1] {
				paramValue = fmt.Sprintf("%v", is.GetObject(i+j))
			} else {
				paramValue = fmt.Sprintf("%v", ins[i+j])
			}
			if j > 1 {
				out.WriteString(fmt.Sprintf(", %v", paramValue))
			} else {
				out.WriteString(fmt.Sprintf("%v", paramValue))
			}
		}
		out.WriteString(fmt.Sprintf("]\tline: %d\n", is.SourceMap[i]))
		i += Instructions[opcode].paramCount + 1
	}

	return out.String()
}
func (is *InstructionSet) define(action int, sourceLine int, params ...interface{}) *anchorReference {
	var ref *anchorReference
	is.Instructions = append(is.Instructions, action)
	is.SourceMap = append(is.SourceMap, sourceLine+1)
	is.Count++
	for _, param := range params {
		switch p := param.(type) {
		case int:
			is.Instructions = append(is.Instructions, p)
		case *anchor:
			is.Instructions = append(is.Instructions, 0)
			ref = &anchorReference{anchor: p, insSet: is, insIndex: is.Count}
		default:
			ci := is.SetConstant(param)
			is.Instructions = append(is.Instructions, ci)
		}
		is.SourceMap = append(is.SourceMap, sourceLine+1)
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

func (is *InstructionSet) elide() {
	length := len(is.Instructions)
	for i := 0; i < length; {
		op := is.Instructions[i]
		switch op {
		case Jump:
			target := is.Instructions[i+1]
			if target > length || is.Instructions[target] == Leave {
				is.Instructions[i] = Leave
				is.Instructions[i+1] = NoOp
				break
			}
			if is.Instructions[target] == Jump {
				is.Instructions[i+1] = is.Instructions[target+1]
				break
			}
		}
		i += 1 + Instructions[op].paramCount
	}
}
