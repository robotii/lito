package parser

import (
	"fmt"
	"strconv"

	"github.com/robotii/lito/compiler/ast"
	"github.com/robotii/lito/compiler/parser/errors"
	"github.com/robotii/lito/compiler/parser/precedence"
	"github.com/robotii/lito/compiler/token"
)

// parseIntegerLiteral parses an integer from the current token
func (p *Parser) parseIntegerLiteral() ast.Expression {
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.error = errors.NewTypeParsingError(p.curToken.Literal, "integer", p.curToken.Line)
		return nil
	}
	return &ast.IntegerLiteral{BaseNode: &ast.BaseNode{Token: p.curToken}, Value: int(value)}
}

func (p *Parser) parseFloatLiteral(integerPart ast.Expression) ast.Expression {
	// Get the fractional part of the token
	p.nextToken()

	floatTok := token.Token{
		Type:    token.Float,
		Literal: fmt.Sprintf("%s.%s", integerPart.String(), p.curToken.Literal),
		Line:    p.curToken.Line,
	}

	value, err := strconv.ParseFloat(floatTok.Literal, 64)
	if err != nil {
		p.error = errors.NewTypeParsingError(floatTok.Literal, "float", p.curToken.Line)
		return nil
	}
	return &ast.FloatLiteral{BaseNode: &ast.BaseNode{Token: floatTok}, Value: float64(value)}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{BaseNode: &ast.BaseNode{Token: p.curToken}, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	value, err := strconv.ParseBool(p.curToken.Literal)
	if err != nil {
		p.error = errors.NewTypeParsingError(p.curToken.Literal, "boolean", p.curToken.Line)
		return nil
	}
	return &ast.BooleanExpression{BaseNode: &ast.BaseNode{Token: p.curToken}, Value: value}
}

func (p *Parser) parseNilExpression() ast.Expression {
	return &ast.NilExpression{BaseNode: &ast.BaseNode{Token: p.curToken}}
}

func (p *Parser) parseHashExpression() ast.Expression {
	return &ast.HashExpression{BaseNode: &ast.BaseNode{Token: p.curToken}, Data: p.parseHashPairs()}
}

func (p *Parser) parseHashPairs() map[string]ast.Expression {
	pairs := map[string]ast.Expression{}

	if p.peekTokenIs(token.RBrace) {
		p.nextToken()
		return pairs
	}

	p.parseHashPair(pairs)

	for p.peekTokenIs(token.Comma) {
		p.nextToken()
		p.parseHashPair(pairs)
	}

	if !p.expectPeek(token.RBrace) {
		// TODO: Error here
		return nil
	}

	return pairs
}

func (p *Parser) parseHashPair(pairs map[string]ast.Expression) {
	var key string
	var value ast.Expression

	p.nextToken()

	switch p.curToken.Type {
	case token.Constant, token.Ident:
		key = p.parseIdentifier().(ast.Variable).ReturnValue()
	case token.String:
		key = p.curToken.Literal
	default:
		p.error = errors.NewTypeParsingError(p.curToken.Literal, "hash key", p.curToken.Line)
		return
	}

	if !p.expectPeek(token.Colon) {
		return
	}

	p.nextToken()
	value = p.parseExpression(precedence.Normal)
	pairs[key] = value
}

func (p *Parser) parseArrayExpression() ast.Expression {
	return &ast.ArrayExpression{BaseNode: &ast.BaseNode{Token: p.curToken}, Elements: p.parseArrayElements()}
}

func (p *Parser) parseArrayElements() []ast.Expression {
	elems := []ast.Expression{}

	if p.peekTokenIs(token.RBracket) {
		p.nextToken()
		return elems
	}

	p.nextToken() // start of first expression
	elems = append(elems, p.parseExpression(precedence.Normal))

	for p.peekTokenIs(token.Comma) {
		p.nextToken() // ","
		p.nextToken() // start of next expression
		elems = append(elems, p.parseExpression(precedence.Normal))
	}

	if !p.expectPeek(token.RBracket) {
		return nil
	}

	return elems
}

func (p *Parser) parseRangeExpression(left ast.Expression) ast.Expression {
	exp := &ast.RangeExpression{
		BaseNode:  &ast.BaseNode{Token: p.curToken},
		Start:     left,
		Exclusive: p.curToken.Type == token.RangeExcl,
	}

	prec := p.curPrecedence()
	p.nextToken()
	exp.End = p.parseExpression(prec)

	return exp
}
