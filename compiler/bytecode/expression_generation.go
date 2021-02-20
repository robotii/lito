package bytecode

import (
	"fmt"

	"github.com/robotii/lito/compiler/ast"
)

var operatorMap = map[string]int{
	"+":  Add,
	"-":  Subtract,
	"<":  Less,
	"<=": LessEqual,
	">":  Greater,
	">=": GreaterEqual,
}

func (g *Generator) compileExpression(is *InstructionSet, exp ast.Expression, scope *scope, table *localTable) {
	if exp == nil {
		return
	}

	sourceLine := exp.Line()
	switch exp := exp.(type) {
	case *ast.Constant:
		if exp.IsNamespace {
			is.define(GetConstantNamespace, sourceLine, exp.Value)
		} else {
			is.define(GetConstant, sourceLine, exp.Value)
		}
	case *ast.InstanceVariable:
		is.define(GetInstanceVariable, sourceLine, exp.Value)
	case *ast.IntegerLiteral:
		is.define(PutInt, sourceLine, exp.Value)
	case *ast.FloatLiteral:
		is.define(PutFloat, sourceLine, exp.Value)
	case *ast.StringLiteral:
		is.define(PutString, sourceLine, exp.Value)
	case *ast.BooleanExpression:
		if exp.Value {
			is.define(PutTrue, sourceLine)
		} else {
			is.define(PutFalse, sourceLine)
		}
	case *ast.NilExpression:
		is.define(PutNull, sourceLine)
	case *ast.RangeExpression:
		g.compileExpression(is, exp.Start, scope, table)
		g.compileExpression(is, exp.End, scope, table)
		if exp.Exclusive {
			is.define(NewRangeExcl, sourceLine)
		} else {
			is.define(NewRange, sourceLine)
		}
	case *ast.ArrayExpression:
		for _, elem := range exp.Elements {
			g.compileExpression(is, elem, scope, table)
		}
		is.define(NewArray, sourceLine, len(exp.Elements))
	case *ast.HashExpression:
		for key, value := range exp.Data {
			is.define(PutString, sourceLine, key)
			g.compileExpression(is, value, scope, table)
		}
		is.define(NewHash, sourceLine, len(exp.Data)*2)
	case *ast.SelfExpression:
		if exp.IsSuper {
			is.define(PutSuper, sourceLine)
		} else {
			is.define(PutSelf, sourceLine)
		}
	case *ast.ArgumentPairExpression:
		g.compileExpression(is, exp.Value, scope, table)
	case *ast.PrefixExpression:
		g.compilePrefixExpression(is, exp, scope, table)
	case *ast.InfixExpression:
		g.compileInfixExpression(is, exp, scope, table)
	case *ast.Identifier:
		g.compileIdentifier(is, exp, scope, table)
	case *ast.AssignExpression:
		g.compileAssignExpression(is, exp, scope, table)
	case *ast.IfExpression:
		g.compileIfExpression(is, exp, scope, table)
	case *ast.YieldExpression:
		g.compileYieldExpression(is, exp, scope, table)
	case *ast.GetBlockExpression:
		g.compileGetBlockExpression(is, exp, scope, table)
	case *ast.HasBlockExpression:
		g.compileHasBlockExpression(is, exp, scope, table)
	case *ast.CallExpression:
		g.compileCallExpression(is, exp, scope, table)
	}
}

func (g *Generator) compileIdentifier(is *InstructionSet, exp *ast.Identifier, scope *scope, table *localTable) {
	index, depth, ok := table.getLocal(exp.Value, table.depth)
	if ok {
		is.define(GetLocal, exp.Line(), depth, index)
		return
	}

	// otherwise it's a method call
	is.define(PutSelf, exp.Line())
	is.define(Send, exp.Line(), exp.Value, 0, nil, nil)
}

func (g *Generator) compileYieldExpression(is *InstructionSet, exp *ast.YieldExpression, scope *scope, table *localTable) {
	is.define(PutSelf, exp.Line())

	for _, arg := range exp.Arguments {
		g.compileExpression(is, arg, scope, table)
	}
	is.define(InvokeBlock, exp.Line(), len(exp.Arguments))
}

func (g *Generator) compileGetBlockExpression(is *InstructionSet, exp *ast.GetBlockExpression, scope *scope, table *localTable) {
	is.define(GetBlock, exp.Line())
}

func (g *Generator) compileHasBlockExpression(is *InstructionSet, exp *ast.HasBlockExpression, scope *scope, table *localTable) {
	is.define(HasBlock, exp.Line())
}

func (g *Generator) compileCallExpression(is *InstructionSet, exp *ast.CallExpression, scope *scope, table *localTable) {
	var blockInfo *InstructionSet
	argSet := &ArgSet{
		names: make([]string, len(exp.Arguments)),
		types: make([]uint8, len(exp.Arguments)),
	}

	// Compile receiver
	g.compileExpression(is, exp.Receiver, scope, table)

	// Compile arguments
	for i, arg := range exp.Arguments {
		switch arg := arg.(type) {
		case *ast.Identifier:
			argSet.setArg(i, arg.Value, NormalArg)
		case *ast.AssignExpression:
			varName := arg.Variables[0].(*ast.Identifier)
			argSet.setArg(i, varName.Value, OptionedArg)
		case *ast.ArgumentPairExpression:
			key := arg.Key.(*ast.Identifier)

			if arg.Value == nil {
				argSet.setArg(i, key.Value, RequiredKeywordArg)
			} else {
				argSet.setArg(i, key.Value, OptionalKeywordArg)
			}
		case *ast.PrefixExpression:
			if arg.Operator == "*" {
				ident, ok := arg.Right.(*ast.Identifier)
				if ok {
					argSet.setArg(i, ident.Value, SplatArg)
				}
			}
		}

		g.compileExpression(is, arg, scope, table)
	}

	// Compile block
	if exp.Block != nil {
		// Inside block should be one level deeper than outside
		newTable := newLocalTable(table.depth + 1)
		newTable.upper = table
		blockIndex := g.blockCounter
		g.blockCounter++
		blockInfo = g.compileBlockArgExpression(blockIndex, exp, scope, newTable)
	}

	if exp.Method == "defer" {
		is.define(Defer, exp.Line(), len(exp.Arguments), blockInfo)
	} else {
		is.define(Send, exp.Line(), exp.Method, len(exp.Arguments), blockInfo, argSet)
	}
}

func (g *Generator) compileAssignExpression(is *InstructionSet, exp *ast.AssignExpression, scope *scope, table *localTable) {
	g.compileExpression(is, exp.Value, scope, table)

	if len(exp.Variables) > 1 {
		is.define(ExpandArray, exp.Line(), len(exp.Variables))
	}

	for i, v := range exp.Variables {
		if v.TokenLiteral() != "_" {

			switch name := v.(type) {
			case *ast.Identifier:
				index, depth := table.setLocal(name.Value, table.depth)

				if exp.Optioned != 0 {
					is.define(SetOptional, exp.Line(), depth, index)
					return
				}

				is.define(SetLocal, exp.Line(), depth, index)
			case *ast.InstanceVariable:
				is.define(SetInstanceVariable, exp.Line(), name.Value)
			case *ast.Constant:
				is.define(SetConstant, exp.Line(), name.Value)
			}
		}

		// Last value should not be popped
		if i != len(exp.Variables)-1 {
			is.define(Pop, exp.Line())
		}
	}
}

func (g *Generator) compileBlockArgExpression(index int, exp *ast.CallExpression, scope *scope, table *localTable) *InstructionSet {
	is := &InstructionSet{Name: fmt.Sprint(index), Type: Block}

	for i := 0; i < len(exp.BlockArguments); i++ {
		table.set(exp.BlockArguments[i].Value)
	}

	g.compileCodeBlock(is, exp.Block, scope, table)
	g.endInstructions(is, exp.Line())
	g.instructionSets = append(g.instructionSets, is)
	return is
}

func (g *Generator) compileIfExpression(is *InstructionSet, exp *ast.IfExpression, scope *scope, table *localTable) {
	anchorLast := &anchor{}

	for _, c := range exp.Conditionals {
		anchorConditional := &anchor{}

		g.compileExpression(is, c.Condition, scope, table)
		bu := is.define(BranchUnless, exp.Line(), anchorConditional)
		g.instructionsWithAnchor = append(g.instructionsWithAnchor, bu)

		if c.Consequence.IsEmpty() {
			is.define(PutNull, exp.Line())
		} else {
			g.compileCodeBlock(is, c.Consequence, scope, table)
		}

		jp := is.define(Jump, exp.Line(), anchorLast)
		anchorConditional.define(is.Count)
		g.instructionsWithAnchor = append(g.instructionsWithAnchor, jp)
	}

	if exp.Alternative == nil {
		// jump over the `putnil` in false case
		is.define(PutNull, exp.Line())
		anchorLast.define(is.Count)
		return
	}

	g.compileCodeBlock(is, exp.Alternative, scope, table)
	anchorLast.define(is.Count)
}

func (g *Generator) compilePrefixExpression(is *InstructionSet, exp *ast.PrefixExpression, scope *scope, table *localTable) {
	switch exp.Operator {
	case "!":
		g.compileExpression(is, exp.Right, scope, table)
		is.define(Send, exp.Line(), exp.Operator, 0, nil, nil)
	case "*":
		g.compileExpression(is, exp.Right, scope, table)
		is.define(SplatArray, exp.Line())
	case "-":
		is.define(PutInt, exp.Line(), 0)
		g.compileExpression(is, exp.Right, scope, table)
		is.define(Send, exp.Line(), exp.Operator, 1, nil, nil)
	case "+":
		g.compileExpression(is, exp.Right, scope, table)
	case "<-", "->":
		is.define(PutSelf, exp.Line())
		g.compileExpression(is, exp.Right, scope, table)
		is.define(Send, exp.Line(), exp.Operator, 1, nil, nil)
	case "&":
		g.compileExpression(is, exp.Right, scope, table)
		is.define(SplatBlock, exp.Line())
	}
}

func (g *Generator) compileInfixExpression(is *InstructionSet, node *ast.InfixExpression, scope *scope, table *localTable) {
	switch node.Operator {
	case "::":
		g.compileExpression(is, node.Left, scope, table)
		g.compileExpression(is, node.Right, scope, table)

	case "&&", "||":
		// We short-circuit these operators
		endAnchor := &anchor{}
		g.compileExpression(is, node.Left, scope, table)
		is.define(Dup, node.Line())
		if node.Operator == "&&" {
			g.instructionsWithAnchor = append(g.instructionsWithAnchor, is.define(BranchUnless, node.Line(), endAnchor))
		} else {
			g.instructionsWithAnchor = append(g.instructionsWithAnchor, is.define(BranchIf, node.Line(), endAnchor))
		}
		is.define(Pop, node.Line())
		g.compileExpression(is, node.Right, scope, table)
		endAnchor.define(len(is.Instructions))

	// Special case the below operators
	case "+", "-", ">", ">=", "<", "<=":
		g.compileExpression(is, node.Left, scope, table)
		g.compileExpression(is, node.Right, scope, table)
		is.define(operatorMap[node.Operator], node.Line(), node.Operator)

	default:
		g.compileExpression(is, node.Left, scope, table)
		g.compileExpression(is, node.Right, scope, table)
		is.define(BinaryOperator, node.Line(), node.Operator)
	}
}
