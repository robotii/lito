package parser

import (
	"github.com/robotii/lito/compiler/ast"
	"github.com/robotii/lito/compiler/parser/precedence"
	"github.com/robotii/lito/compiler/token"
)

func (p *Parser) parseSwitchExpression() ast.Expression {

	ie := &ast.IfExpression{BaseNode: &ast.BaseNode{Token: p.curToken}}
	ie.Conditionals = p.parseSwitchConditionals()

	if p.curTokenIs(token.Default) {
		ie.Alternative = p.parseBlockStatement(token.RBrace)
		ie.Alternative.KeepLastValue()
	}

	return ie
}

// case expression parsing helpers
func (p *Parser) parseSwitchConditionals() []*ast.ConditionalExpression {
	var ce []*ast.ConditionalExpression
	var base ast.Expression

	p.acceptBlock = false
	if p.peekTokenIs(token.LBrace) {
		base = &ast.BooleanExpression{BaseNode: &ast.BaseNode{Token: token.Token{Type: token.True, Literal: "true", Line: p.curToken.Line}}, Value: true}
		p.expectPeek(token.LBrace)
	} else {
		p.nextToken()
		base = p.parseExpression(precedence.Normal)
		p.expectPeek(token.LBrace)
	}

	p.acceptBlock = true
	p.expectPeek(token.Case)

	for p.curTokenIs(token.Case) {
		ce = append(ce, p.parseSwitchConditional(base))
	}

	return ce
}

func (p *Parser) parseSwitchConditional(base ast.Expression) *ast.ConditionalExpression {
	ce := &ast.ConditionalExpression{BaseNode: &ast.BaseNode{Token: p.curToken}}
	p.nextToken()

	ce.Condition = p.parseSwitchCondition(base)
	ce.Consequence = p.parseBlockStatement(token.Case, token.Default, token.RBrace)
	ce.Consequence.KeepLastValue()

	return ce
}

func (p *Parser) parseSwitchCondition(base ast.Expression) *ast.InfixExpression {
	first := p.parseExpression(precedence.Normal)
	infix := newInfixExpression(base, token.Token{Type: token.IsSame, Literal: token.IsSame}, first)

	for p.peekTokenIs(token.Comma) {
		p.nextToken()
		p.nextToken()

		right := p.parseExpression(precedence.Normal)
		rightInfix := newInfixExpression(base, token.Token{Type: token.IsSame, Literal: token.IsSame}, right)
		infix = newInfixExpression(infix, token.Token{Type: token.Or, Literal: token.Or}, rightInfix)
	}

	return infix
}
