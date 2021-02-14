package parser

import (
	"github.com/robotii/lito/compiler/ast"
	"github.com/robotii/lito/compiler/parser/precedence"
	"github.com/robotii/lito/compiler/token"
)

func (p *Parser) parseIfExpression() ast.Expression {
	ie := &ast.IfExpression{BaseNode: &ast.BaseNode{Token: p.curToken}}
	// parse if and elsif expressions
	ie.Conditionals = p.parseConditionalExpressions()

	// curToken is now ELSE or RBRACE
	if p.curTokenIs(token.RBrace) && p.peekTokenIs(token.Else) {
		p.nextToken()
	}
	if p.curTokenIs(token.Else) {
		p.expectPeek(token.LBrace)
		ie.Alternative = p.parseBlockStatement(token.RBrace)
		ie.Alternative.KeepLastValue()
	}

	return ie
}

// infix expression parsing helpers
func (p *Parser) parseConditionalExpressions() []*ast.ConditionalExpression {
	// first conditional expression should start with if
	p.acceptBlock = false
	cs := []*ast.ConditionalExpression{p.parseConditionalExpression()}
	p.acceptBlock = true

	if p.curTokenIs(token.RBrace) && p.peekTokenIs(token.ElsIf) {
		p.nextToken()
	}
	for p.curTokenIs(token.ElsIf) {
		p.acceptBlock = false
		cs = append(cs, p.parseConditionalExpression())
		p.acceptBlock = true
		if p.curTokenIs(token.RBrace) && p.peekTokenIs(token.ElsIf) {
			p.nextToken()
		}
	}

	return cs
}

func (p *Parser) parseConditionalExpression() *ast.ConditionalExpression {
	ce := &ast.ConditionalExpression{BaseNode: &ast.BaseNode{Token: p.curToken}}
	p.nextToken()
	ce.Condition = p.parseExpression(precedence.Normal)
	p.expectPeek(token.LBrace)
	ce.Consequence = p.parseBlockStatement(token.RBrace)
	ce.Consequence.KeepLastValue()

	return ce
}
