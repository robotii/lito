package bytecode

import (
	"github.com/robotii/lito/compiler/ast"
)

// These constants are enums that represent argument's types
const (
	NormalArg uint8 = iota
	OptionedArg
	SplatArg
	RequiredKeywordArg
	OptionalKeywordArg
)

func (g *Generator) compileStatements(stmts []ast.Statement, scope *scope, table *localTable) {
	is := &InstructionSet{Type: Program, Name: Program}

	for _, statement := range stmts {
		g.compileStatement(is, statement, scope, table)
	}

	// empty input so no statement is given
	if len(stmts) == 0 {
		return
	}

	g.endInstructions(is, stmts[len(stmts)-1].Line())
	g.instructionSets = append(g.instructionSets, is)
}

func (g *Generator) compileStatement(is *InstructionSet, statement ast.Statement, scope *scope, table *localTable) {
	switch stmt := statement.(type) {
	case *ast.ExpressionStatement:
		g.compileExpression(is, stmt.Expression, scope, table)
		if !g.REPL && stmt.Expression.IsStmt() {
			is.define(Pop, statement.Line())
		}
	case *ast.DefStatement:
		g.compileDefStmt(is, stmt, scope)
	case *ast.ClassStatement:
		g.compileClassStmt(is, stmt, scope, table)
		if stmt.SuperClass != nil {
			is.define(Pop, statement.Line())
		}
	case *ast.ModuleStatement:
		g.compileModuleStmt(is, stmt, scope)
	case *ast.ReturnStatement:
		g.compileExpression(is, stmt.ReturnValue, scope, table)
		g.endInstructions(is, stmt.Line())
	case *ast.WhileStatement:
		g.compileWhileStmt(is, stmt, scope, table)
	case *ast.ContinueStatement:
		g.compileContinueStatement(is, stmt, scope)
	case *ast.BreakStatement:
		g.compileBreakStatement(is, stmt, scope)
	}
}

func (g *Generator) compileWhileStmt(is *InstructionSet, stmt *ast.WhileStatement, scope *scope, table *localTable) {
	anchor1 := &anchor{}
	breakAnchor := &anchor{}

	jp := is.define(Jump, stmt.Line(), anchor1)
	g.instructionsWithAnchor = append(g.instructionsWithAnchor, jp)

	anchor2 := &anchor{is.Count}

	// we need to save the achors for this scope
	outerNextAnchor := scope.anchors["continue"]
	outerBreakAnchor := scope.anchors["break"]

	scope.anchors["continue"] = anchor1
	scope.anchors["break"] = breakAnchor

	g.compileCodeBlock(is, stmt.Body, scope, table)

	// replace
	scope.anchors["continue"] = outerNextAnchor
	scope.anchors["break"] = outerBreakAnchor
	anchor1.define(is.Count)

	g.compileExpression(is, stmt.Condition, scope, table)

	bi := is.define(BranchIf, stmt.Line(), anchor2)
	g.instructionsWithAnchor = append(g.instructionsWithAnchor, bi)
	breakAnchor.define(is.Count)
}

func (g *Generator) compileContinueStatement(is *InstructionSet, stmt ast.Statement, scope *scope) {
	// TODO: Need to ensure we are in loop
	if scope.anchors["continue"] != nil {
		if is.Type == Block {
			is.define(Leave, stmt.Line())
		}
		jp := is.define(Jump, stmt.Line(), scope.anchors["continue"])
		g.instructionsWithAnchor = append(g.instructionsWithAnchor, jp)
	} else {
		is.define(Leave, stmt.Line())
	}
}

func (g *Generator) compileBreakStatement(is *InstructionSet, stmt ast.Statement, scope *scope) {
	if scope.anchors["break"] != nil {
		if is.Type == Block {
			is.define(Break, stmt.Line())
		}
		jp := is.define(Jump, stmt.Line(), scope.anchors["break"])
		g.instructionsWithAnchor = append(g.instructionsWithAnchor, jp)
	} else {
		is.define(Break, stmt.Line())
	}
}

func (g *Generator) compileClassStmt(is *InstructionSet, stmt *ast.ClassStatement, scope *scope, table *localTable) {
	originalScope := scope
	scope = newScope()

	// compile class's content
	newIS := &InstructionSet{Name: stmt.Name.Value, Type: Class}

	g.compileCodeBlock(newIS, stmt.Body, scope, scope.localTable)
	newIS.define(Leave, stmt.Line())
	g.instructionSets = append(g.instructionSets, newIS)

	is.define(PutSelf, stmt.Line())

	if stmt.SuperClass != nil {
		g.compileExpression(is, stmt.SuperClass, originalScope, table)
		is.define(DefClass, stmt.Line(), "class", stmt.Name.Value, newIS, stmt.SuperClassName)
	} else {
		is.define(DefClass, stmt.Line(), "class", stmt.Name.Value, newIS, NoSuperClass)
	}

	is.define(Pop, stmt.Line())

}

func (g *Generator) compileModuleStmt(is *InstructionSet, stmt *ast.ModuleStatement, scope *scope) {
	scope = newScope()
	newIS := &InstructionSet{Name: stmt.Name.Value, Type: Class}

	g.compileCodeBlock(newIS, stmt.Body, scope, scope.localTable)
	newIS.define(Leave, stmt.Line())
	g.instructionSets = append(g.instructionSets, newIS)
	is.define(PutSelf, stmt.Line())
	is.define(DefClass, stmt.Line(), "module", stmt.Name.Value, newIS, NoSuperClass)
	is.define(Pop, stmt.Line())
}

func (g *Generator) compileDefStmt(is *InstructionSet, stmt *ast.DefStatement, scope *scope) {
	originalScope := scope
	scope = newScope()

	// compile method definition's content
	newIS := &InstructionSet{
		Name: stmt.Name.Value,
		Type: Method,
		ArgTypes: ArgSet{
			names: make([]string, len(stmt.Parameters)),
			types: make([]uint8, len(stmt.Parameters)),
		},
	}

	for i, parameter := range stmt.Parameters {
		switch exp := parameter.(type) {
		case *ast.Identifier:
			scope.localTable.setLocal(exp.Value, scope.localTable.depth)

			newIS.ArgTypes.setArg(i, exp.Value, NormalArg)
		case *ast.AssignExpression:
			exp.Optioned = 1

			v := exp.Variables[0]
			varName := v.(*ast.Identifier)
			g.compileAssignExpression(newIS, exp, scope, scope.localTable)

			newIS.ArgTypes.setArg(i, varName.Value, OptionedArg)
		case *ast.PrefixExpression:
			if exp.Operator != "*" {
				continue
			}

			ident := exp.Right.(*ast.Identifier)

			// Set default value to an empty array
			index, depth := scope.localTable.setLocal(ident.Value, scope.localTable.depth)
			newIS.define(NewArray, exp.Line(), 0)
			newIS.define(SetOptional, exp.Line(), depth, index)

			newIS.ArgTypes.setArg(i, ident.Value, SplatArg)
		case *ast.ArgumentPairExpression:
			key := exp.Key.(*ast.Identifier)
			index, depth := scope.localTable.setLocal(key.Value, scope.localTable.depth)

			if exp.Value != nil {
				g.compileExpression(newIS, exp.Value, scope, scope.localTable)
				newIS.define(SetOptional, exp.Line(), depth, index)
				newIS.ArgTypes.setArg(i, key.Value, OptionalKeywordArg)
			} else {
				newIS.ArgTypes.setArg(i, key.Value, RequiredKeywordArg)
			}
		}
	}

	if len(stmt.BlockStatement.Statements) == 0 {
		newIS.define(PutNull, stmt.Line())
	} else {
		g.compileCodeBlock(newIS, stmt.BlockStatement, scope, scope.localTable)
	}

	g.endInstructions(newIS, stmt.Line())
	g.instructionSets = append(g.instructionSets, newIS)

	switch stmt.Receiver.(type) {
	case nil:
		is.define(PutSelf, stmt.Line())
		is.define(DefMethod, stmt.Line(), len(stmt.Parameters), stmt.Name.Value, newIS)
	default:
		g.compileExpression(is, stmt.Receiver, originalScope, originalScope.localTable)
		is.define(DefMetaMethod, stmt.Line(), len(stmt.Parameters), stmt.Name.Value, newIS)
	}
}
