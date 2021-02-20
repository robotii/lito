package bytecode

import (
	"github.com/robotii/lito/compiler/ast"
)

type scope struct {
	program    *ast.Program
	localTable *localTable
	anchors    map[string]*anchor
}

func newScope() *scope {
	return &scope{localTable: newLocalTable(0), anchors: make(map[string]*anchor)}
}

// Generator contains program's AST and will store generated instruction sets
type Generator struct {
	REPL                   bool
	instructionSets        []*InstructionSet
	blockCounter           int
	scope                  *scope
	instructionsWithAnchor []*anchorReference
}

// NewGenerator initializes new Generator with complete AST tree.
func NewGenerator() *Generator {
	return &Generator{}
}

// ResetMethodInstructionSets clears generator's method instruction sets
func (g *Generator) ResetMethodInstructionSets() {
	iSets := g.instructionSets
	g.instructionSets = []*InstructionSet{}
	// We only copy back the blocks, as the rest are "spent"
	for _, set := range iSets {
		if set.Type == Block {
			g.instructionSets = append(g.instructionSets, set)
		}
	}
}

// Index returns the current count of the instruction sets
func (g *Generator) Index() int {
	return len(g.instructionSets)
}

// InitTopLevelScope sets generator's scope with program node, which means it's the top level scope
func (g *Generator) InitTopLevelScope(program *ast.Program) {
	g.scope = &scope{program: program, localTable: newLocalTable(0), anchors: make(map[string]*anchor)}
}

// GenerateInstructions returns compiled instructions
func (g *Generator) GenerateInstructions(stmts []ast.Statement) []*InstructionSet {
	g.compileStatements(stmts, g.scope, g.scope.localTable)
	// Use anchor's exact position to replace anchor obj
	for _, i := range g.instructionsWithAnchor {
		if i != nil && i.anchor != nil && i.insSet != nil {
			i.insSet.Instructions[i.insIndex].Params[0] = i.anchor.line
		}
	}
	// Reset the anchor list
	g.instructionsWithAnchor = nil
	// Perform some optimisations on the bytecode
	for _, i := range g.instructionSets {
		i.elide()
	}
	return g.instructionSets
}

func (g *Generator) compileCodeBlock(is *InstructionSet, stmt *ast.BlockStatement, scope *scope, table *localTable) {
	for _, s := range stmt.Statements {
		g.compileStatement(is, s, scope, table)
	}
}

func (g *Generator) endInstructions(is *InstructionSet, sourceLine int) {
	if g.REPL && is.Name == Program {
		return
	}
	is.define(Leave, sourceLine)
}
