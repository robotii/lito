package parser

import (
	"fmt"

	"github.com/robotii/lito/compiler/ast"
	"github.com/robotii/lito/compiler/parser/arguments"
	"github.com/robotii/lito/compiler/parser/errors"
	"github.com/robotii/lito/compiler/parser/precedence"
	"github.com/robotii/lito/compiler/parser/states"
	"github.com/robotii/lito/compiler/token"
)

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.Return:
		return p.parseReturnStatement()
	case token.Def:
		return p.parseDefMethodStatement()
	case token.Comment:
		return nil
	case token.While:
		return p.parseWhileStatement()
	case token.Class:
		return p.parseClassStatement()
	case token.Module:
		return p.parseModuleStatement()
	case token.Continue:
		return &ast.ContinueStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}
	case token.Break:
		return &ast.BreakStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}
	default:
		exp := p.parseExpressionStatement()

		// If parseExpressionStatement got error exp.Expression would be nil
		if exp.Expression != nil {
			// In REPL mode everything should return a value.
			if p.Mode == REPLMode {
				exp.Expression.MarkAsExp()
			} else {
				exp.Expression.MarkAsStmt()
			}
		}

		return exp
	}
}

func (p *Parser) parseDefMethodStatement() *ast.DefStatement {
	var params []ast.Expression
	stmt := &ast.DefStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}

	p.nextToken()

	if p.IsNotDefMethodToken() {
		msg := fmt.Sprintf("Invalid method name: %s. Line: %d", p.curToken.Literal, p.curToken.Line)
		p.error = errors.InitError(msg, errors.MethodDefinitionError)
		return nil
	}
	// Method has specific receiver like `def self.foo` or `def bar.foo`
	if p.peekTokenIs(token.Dot) {
		switch p.curToken.Type {
		case token.Ident:
			stmt.Receiver = p.parseIdentifier()
		case token.InstanceVariable:
			stmt.Receiver = p.parseInstanceVariable()
		case token.Constant:
			stmt.Receiver = p.parseConstant()
		case token.Self:
			stmt.Receiver = &ast.SelfExpression{BaseNode: &ast.BaseNode{Token: p.curToken}}
		default:
			msg := fmt.Sprintf("Invalid method receiver: %s. Line: %d", p.curToken.Literal, p.curToken.Line)
			p.error = errors.InitError(msg, errors.MethodDefinitionError)
		}

		p.nextToken() // .
		if !p.expectPeek(token.Ident) {
			return nil
		}
	}

	stmt.Name = &ast.Identifier{BaseNode: &ast.BaseNode{Token: p.curToken}, Value: p.curToken.Literal}

	if p.peekTokenIs(token.Assign) {
		stmt.Name.Value += "="
		p.nextToken()
	}

	if p.peekTokenIs(token.Ident) && p.peekTokenAtSameLine() { // def foo x, next token is x and at same line
		msg := fmt.Sprintf("Please add parentheses around method \"%s\"'s parameters. Line: %d", stmt.Name.Value, p.curToken.Line)
		p.error = errors.InitError(msg, errors.MethodDefinitionError)
	}

	if p.peekTokenIs(token.LParen) && p.peekTokenAtSameLine() {
		p.nextToken()

		switch p.peekToken.Type {
		case token.RParen:
			params = []ast.Expression{}
		default:
			params = p.parseParameters()
		}

		if p.IsNotParamsToken() {
			return nil
		}

		if !p.expectPeek(token.RParen) {
			return nil
		}
	} else {
		params = []ast.Expression{}
	}

	stmt.Parameters = params
	p.expectPeek(token.LBrace)
	stmt.BlockStatement = p.parseBlockStatement(token.RBrace)
	stmt.BlockStatement.KeepLastValue()

	return stmt
}

func (p *Parser) parseParameters() []ast.Expression {
	p.fsm.State(states.ParsingMethodParam)
	params := []ast.Expression{}

	p.nextToken()

	if p.IsNotParamsToken() {
		msg := fmt.Sprintf("Invalid parameters: %s. Line: %d", p.curToken.Literal, p.curToken.Line)
		p.error = errors.InitError(msg, errors.MethodDefinitionError)
		return nil
	}

	param := p.parseExpression(precedence.Normal)
	params = append(params, param)

	for p.peekTokenIs(token.Comma) {
		p.nextToken()
		p.nextToken()

		if p.IsNotParamsToken() {
			msg := fmt.Sprintf("Invalid parameters: %s. Line: %d", p.curToken.Literal, p.curToken.Line)
			p.error = errors.InitError(msg, errors.MethodDefinitionError)
			return nil
		}

		if p.curTokenIs(token.Asterisk) && !p.peekTokenIs(token.Ident) {
			p.expectPeek(token.Ident)
			break
		}

		param := p.parseExpression(precedence.Normal)
		params = append(params, param)
	}

	p.checkMethodParameters(params)

	p.fsm.State(states.Normal)
	return params
}

func (p *Parser) checkMethodParameters(params []ast.Expression) {
	argState := arguments.NormalArg

	checkedParams := []ast.Expression{}

	for _, param := range params {
		switch exp := param.(type) {
		case *ast.Identifier:
			switch argState {
			case arguments.OptionedArg:
				p.error = errors.NewArgumentError(arguments.NormalArg, arguments.OptionedArg, exp.Value, p.curToken.Line)
			case arguments.RequiredKeywordArg:
				p.error = errors.NewArgumentError(arguments.NormalArg, arguments.RequiredKeywordArg, exp.Value, p.curToken.Line)
			case arguments.OptionalKeywordArg:
				p.error = errors.NewArgumentError(arguments.NormalArg, arguments.OptionalKeywordArg, exp.Value, p.curToken.Line)
			case arguments.SplatArg:
				p.error = errors.NewArgumentError(arguments.NormalArg, arguments.SplatArg, exp.Value, p.curToken.Line)
			}
		case *ast.AssignExpression:
			switch argState {
			case arguments.RequiredKeywordArg:
				p.error = errors.NewArgumentError(arguments.OptionedArg, arguments.RequiredKeywordArg, exp.String(), p.curToken.Line)
			case arguments.OptionalKeywordArg:
				p.error = errors.NewArgumentError(arguments.OptionedArg, arguments.OptionalKeywordArg, exp.String(), p.curToken.Line)
			case arguments.SplatArg:
				p.error = errors.NewArgumentError(arguments.OptionedArg, arguments.SplatArg, exp.String(), p.curToken.Line)
			}
			argState = arguments.OptionedArg
		case *ast.ArgumentPairExpression:
			if exp.Value == nil {
				switch argState {
				case arguments.OptionalKeywordArg:
					p.error = errors.NewArgumentError(arguments.RequiredKeywordArg, arguments.OptionalKeywordArg, exp.String(), p.curToken.Line)
				case arguments.SplatArg:
					p.error = errors.NewArgumentError(arguments.RequiredKeywordArg, arguments.SplatArg, exp.String(), p.curToken.Line)
				}

				argState = arguments.RequiredKeywordArg
			} else {
				switch argState {
				case arguments.SplatArg:
					p.error = errors.NewArgumentError(arguments.OptionalKeywordArg, arguments.SplatArg, exp.String(), p.curToken.Line)
				}

				argState = arguments.OptionalKeywordArg
			}
		case *ast.PrefixExpression:
			switch argState {
			case arguments.SplatArg:
				msg := fmt.Sprintf("Can't define splat argument more than once. Line: %d", p.curToken.Line)
				p.error = errors.InitError(msg, errors.ArgumentError)
			}
			argState = arguments.SplatArg
		}

		if p.error != nil {
			break
		}

		if paramDuplicated(checkedParams, param) {
			msg := fmt.Sprintf("Duplicate argument name: \"%s\". Line: %d", getArgName(param), p.curToken.Line)
			p.error = errors.InitError(msg, errors.ArgumentError)
		} else {
			checkedParams = append(checkedParams, param)
		}
	}
}

func (p *Parser) parseClassStatement() *ast.ClassStatement {
	stmt := &ast.ClassStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}

	if !p.expectPeek(token.Constant) {
		return nil
	}

	stmt.Name = &ast.Constant{BaseNode: &ast.BaseNode{Token: p.curToken}, Value: p.curToken.Literal}

	// See if there is any inheritance
	if p.peekTokenIs(token.LT) {
		p.nextToken() // <
		p.nextToken() // Inherited class like 'Bar'
		stmt.SuperClass = p.parseExpression(precedence.Normal)

		switch exp := stmt.SuperClass.(type) {
		case *ast.InfixExpression:
			stmt.SuperClassName = exp.Right.(*ast.Constant).Value
		case *ast.Constant:
			stmt.SuperClassName = exp.Value
		}
	}

	p.expectPeek(token.LBrace)
	stmt.Body = p.parseBlockStatement(token.RBrace)

	return stmt
}

func (p *Parser) parseModuleStatement() *ast.ModuleStatement {
	stmt := &ast.ModuleStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}

	if !p.expectPeek(token.Constant) {
		return nil
	}

	stmt.Name = &ast.Constant{BaseNode: &ast.BaseNode{Token: p.curToken}, Value: p.curToken.Literal}
	p.expectPeek(token.LBrace)
	stmt.Body = p.parseBlockStatement(token.RBrace)
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}

	// If we have a token on the same line, this will be the return value
	// Otherwise we will have nil as the return value
	if !p.peekTokenAtSameLine() {
		stmt.ReturnValue = &ast.NilExpression{BaseNode: &ast.BaseNode{Token: p.curToken}}
		return stmt
	}

	p.nextToken()
	stmt.ReturnValue = p.parseExpression(precedence.Normal)
	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}
	// If we have an identifier or instance variable, we need to check
	// for multiple variable assignment as well as a method call
	if p.curTokenIs(token.Ident) || p.curTokenIs(token.InstanceVariable) {
		stmt.Expression = p.parseExpression(precedence.Lowest)
	} else {
		stmt.Expression = p.parseExpression(precedence.Normal)
	}

	return stmt
}

// parseBlockStatement parses a list of statements and returns when the next token
// is one of the end tokens supplied.
func (p *Parser) parseBlockStatement(endTokens ...token.Type) *ast.BlockStatement {
	p.acceptBlock = true
	bs := &ast.BlockStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}
	bs.Statements = []ast.Statement{}

	if p.curTokenIs(token.RBrace) {
		msg := fmt.Sprintf("syntax error, unexpected '%s' Line: %d", p.curToken.Literal, p.curToken.Line)
		p.error = errors.InitError(msg, errors.SyntaxError)
		return bs
	}

	p.nextToken()

	if p.curTokenIs(token.Semicolon) {
		p.nextToken()
	}

	for {
		for _, t := range endTokens {
			if p.curTokenIs(t) {
				return bs
			}
		}

		if p.curTokenIs(token.EOF) {
			p.error = errors.InitError("Unexpected EOF", errors.UnexpectedEOFError)
			return bs
		}
		stmt := p.parseStatement()
		if p.error != nil {
			return bs
		}

		if stmt != nil {
			bs.Statements = append(bs.Statements, stmt)
		}
		p.nextToken()
	}
}

func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	ws := &ast.WhileStatement{BaseNode: &ast.BaseNode{Token: p.curToken}}

	p.nextToken()
	// Prevent consuming while block as argument.
	p.acceptBlock = false

	oldState := p.fsm.Current()
	p.fsm.State(states.ParsingFuncCall)

	ws.Condition = p.parseExpression(precedence.Normal)

	p.fsm.State(oldState)
	p.acceptBlock = true

	p.expectPeek(token.LBrace)
	ws.Body = p.parseBlockStatement(token.RBrace)

	return ws
}

func paramDuplicated(params []ast.Expression, param ast.Expression) bool {
	for _, p := range params {
		if getArgName(param) == getArgName(p) {
			return true
		}
	}
	return false
}

func getArgName(exp ast.Expression) string {
	switch exp := exp.(type) {
	case *ast.AssignExpression:
		return exp.Variables[0].TokenLiteral()
	case *ast.ArgumentPairExpression:
		return exp.Key.(*ast.Identifier).Value
	}
	return exp.TokenLiteral()
}
