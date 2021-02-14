package compiler

import (
	"fmt"

	"github.com/robotii/lito/compiler/bytecode"
	"github.com/robotii/lito/compiler/lexer"
	"github.com/robotii/lito/compiler/parser"
)

// CompileToInstructions compiles input source code into instruction set data structures
func CompileToInstructions(input string, pm parser.Mode) ([]*bytecode.InstructionSet, error) {
	l := lexer.New(input)
	p := parser.New(l, pm)
	program, err := p.ParseProgram()
	if err != nil {
		return nil, fmt.Errorf(err.Message)
	}
	g := bytecode.NewGenerator()
	g.InitTopLevelScope(program)
	return g.GenerateInstructions(program.Statements), nil
}
