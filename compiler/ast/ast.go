package ast

import (
	"strings"

	"github.com/robotii/lito/compiler/token"
)

// BaseNode holds the attribute every expression or statement should have
type BaseNode struct {
	Token  token.Token
	isStmt bool
}

// Line returns node's token's line number
func (b *BaseNode) Line() int {
	return b.Token.Line
}

// IsExp returns if current node should be considered as an expression
func (b *BaseNode) IsExp() bool {
	return !b.isStmt
}

// IsStmt returns if current node should be considered as a statement
func (b *BaseNode) IsStmt() bool {
	return b.isStmt
}

// MarkAsStmt marks current node to be statement
func (b *BaseNode) MarkAsStmt() {
	b.isStmt = true
}

// MarkAsExp marks current node to be expression
func (b *BaseNode) MarkAsExp() {
	b.isStmt = false
}

type node interface {
	TokenLiteral() string
	String() string
	Line() int
	IsExp() bool
	IsStmt() bool
	MarkAsStmt()
	MarkAsExp()
}

// Statement is the AST node for a statement
type Statement interface {
	node
	statementNode()
}

// Expression is the AST node for an expression
type Expression interface {
	node
	expressionNode()
}

// Program is the root node of entire AST
type Program struct {
	Statements []Statement
}

// TokenLiteral returns the string literal for this AST node
func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}

	return ""
}

// String returns the string representation of the program
func (p *Program) String() string {
	var out strings.Builder

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}
