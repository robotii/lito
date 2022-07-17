package parser

import (
	"github.com/robotii/lito/compiler/ast"
	"github.com/robotii/lito/compiler/parser/arguments"
	"github.com/robotii/lito/compiler/parser/precedence"
	"github.com/robotii/lito/compiler/parser/states"
	"github.com/robotii/lito/compiler/token"
)

func (p *Parser) parseCallExpressionWithoutReceiver(receiver ast.Expression) ast.Expression {
	methodToken := receiver.(*ast.Identifier).Token

	exp := &ast.CallExpression{BaseNode: &ast.BaseNode{}}

	oldState := p.fsm.Current()
	p.fsm.State(states.ParsingFuncCall)
	// set receiver to self
	self := &ast.SelfExpression{BaseNode: &ast.BaseNode{
		Token: token.Token{Type: token.Self, Literal: "self", Line: p.curToken.Line}}}

	exp.Token = methodToken
	exp.Receiver = self
	exp.Method = methodToken.Literal

	if p.curTokenIs(token.LParen) {
		exp.Arguments = p.parseCallArgumentsWithParens()
	} else if p.peekTokenIs(token.LBrace) && p.acceptBlock && p.peekTokenAtSameLine() {
		exp.Arguments = []ast.Expression{}
	} else if arguments.Tokens[p.peekToken.Type] && p.peekTokenAtSameLine() {
		p.nextToken()
		exp.Arguments = p.parseCallArguments()
	} else {
		exp.Arguments = []ast.Expression{}
	}

	p.fsm.State(oldState)

	if p.peekTokenIs(token.LBrace) && p.acceptBlock && p.peekTokenAtSameLine() {
		p.parseBlockArgument(exp)
	}

	return exp
}

func (p *Parser) parseCallExpressionWithReceiver(receiver ast.Expression) ast.Expression {
	exp := &ast.CallExpression{BaseNode: &ast.BaseNode{}}

	oldState := p.fsm.Current()
	p.fsm.State(states.ParsingFuncCall)

	// check if method name is identifier
	if !p.expectPeek(token.Ident) {
		return nil
	}

	exp.Token = p.curToken
	exp.Receiver = receiver
	exp.Method = p.curToken.Literal

	if p.peekTokenAtSameLine() {
		switch p.peekToken.Type {
		case token.LParen:
			p.nextToken()
			exp.Arguments = p.parseCallArgumentsWithParens()
		case token.Assign:
			exp.Method += "="
			p.nextToken()
			p.nextToken()
			exp.Arguments = append(exp.Arguments, p.parseExpression(precedence.Normal))
		case token.LBrace:
			exp.Arguments = []ast.Expression{}
		default:
			if arguments.Tokens[p.peekToken.Type] {
				p.nextToken()
				exp.Arguments = p.parseCallArguments()
			} else {
				exp.Arguments = []ast.Expression{}
			}
		}
	} else {
		exp.Arguments = []ast.Expression{}
	}

	p.fsm.State(oldState)

	// Parse block
	if p.peekTokenIs(token.LBrace) && p.acceptBlock && p.peekTokenAtSameLine() {
		p.parseBlockArgument(exp)
	}

	return exp
}

func (p *Parser) parseCallArgumentsWithParens() []ast.Expression {
	args := []ast.Expression{}

	if p.peekTokenIs(token.RParen) {
		p.nextToken() // ')'
		return args
	}

	p.nextToken() // move to first argument token

	args = p.parseCallArguments()

	if !p.expectPeek(token.RParen) {
		return nil
	}

	return args
}

func (p *Parser) parseCallArguments() []ast.Expression {
	args := []ast.Expression{}

	if p.curTokenIs(token.EOF) {
		return args
	}

	args = append(args, p.parseExpression(precedence.Normal))

	for p.peekTokenIs(token.Comma) {
		p.nextToken() // ","
		p.nextToken() // start of next expression
		args = append(args, p.parseExpression(precedence.Normal))
	}

	return args
}

func (p *Parser) parseBlockArgument(exp *ast.CallExpression) {
	p.nextToken()

	// Parse block arguments
	if p.peekTokenIs(token.Bar) {
		var params []*ast.Identifier

		p.nextToken()
		p.nextToken()

		param := &ast.Identifier{BaseNode: &ast.BaseNode{Token: p.curToken}, Value: p.curToken.Literal}
		params = append(params, param)

		for p.peekTokenIs(token.Comma) {
			p.nextToken()
			p.nextToken()
			param := &ast.Identifier{BaseNode: &ast.BaseNode{Token: p.curToken}, Value: p.curToken.Literal}
			params = append(params, param)
		}

		if !p.expectPeek(token.Bar) {
			return
		}

		exp.BlockArguments = params
	}

	exp.Block = p.parseBlockStatement(token.RBrace)
	exp.Block.KeepLastValue()
}
