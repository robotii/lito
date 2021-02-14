package ast

import (
	"strings"
)

// Variable interface represents assignable nodes,
// currently these are Identifier, InstanceVariable and Constant
type Variable interface {
	variableNode()
	ReturnValue() string
	Expression
}

// MultiVariableExpression is not really an expression, it's just a container that holds multiple Variables
type MultiVariableExpression struct {
	*BaseNode
	Variables []Expression
}

func (m *MultiVariableExpression) expressionNode() {}

// TokenLiteral returns the token for a multvariable expression
func (m *MultiVariableExpression) TokenLiteral() string {
	return ""
}
func (m *MultiVariableExpression) String() string {
	var out strings.Builder
	var variables []string

	for _, v := range m.Variables {
		variables = append(variables, v.String())
	}

	out.WriteString(strings.Join(variables, ", "))

	return out.String()
}

// Identifier represents an identifier
type Identifier struct {
	*BaseNode
	Value string
}

func (i *Identifier) variableNode() {}

// ReturnValue returns the value of the identifier
func (i *Identifier) ReturnValue() string {
	return i.Value
}
func (i *Identifier) expressionNode() {}

// TokenLiteral returns the token for an identifier expression
func (i *Identifier) TokenLiteral() string {
	return i.Token.Literal
}
func (i *Identifier) String() string {
	return i.Value
}

// InstanceVariable represents an instance variable
type InstanceVariable struct {
	*BaseNode
	Value string
}

func (iv *InstanceVariable) variableNode() {}

// ReturnValue returns the value of the instance variable
func (iv *InstanceVariable) ReturnValue() string {
	return iv.Value
}
func (iv *InstanceVariable) expressionNode() {}

// TokenLiteral returns the token for a instance variable expression
func (iv *InstanceVariable) TokenLiteral() string {
	return iv.Token.Literal
}
func (iv *InstanceVariable) String() string {
	return iv.Value
}

// Constant represents a constant
type Constant struct {
	*BaseNode
	Value       string
	IsNamespace bool
}

func (c *Constant) variableNode() {}

// ReturnValue returns the value of the constant
func (c *Constant) ReturnValue() string {
	return c.Value
}
func (c *Constant) expressionNode() {}

// TokenLiteral returns the token for a constant expression
func (c *Constant) TokenLiteral() string {
	return c.Token.Literal
}
func (c *Constant) String() string {
	return c.Value
}
