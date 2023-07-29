package precedence

import "github.com/robotii/lito/compiler/token"

// Constants for denoting precedence
const (
	_ = iota
	Lowest
	Normal
	Assign
	Logic
	Range
	Equals
	Compare
	Sum
	Product
	BangPrefix
	Index
	Call
	MinusPrefix
)

// LookupTable maps token to its corresponding precedence
var LookupTable = map[token.Type]int{
	token.Eq:                 Equals,
	token.NotEq:              Equals,
	token.IsSame:             Equals,
	token.IsNotSame:          Equals,
	token.Match:              Compare,
	token.LT:                 Compare,
	token.LTE:                Compare,
	token.GT:                 Compare,
	token.GTE:                Compare,
	token.And:                Logic,
	token.Or:                 Logic,
	token.Range:              Range,
	token.RangeExcl:          Range,
	token.Plus:               Sum,
	token.Minus:              Sum,
	token.Modulo:             Sum,
	token.Slash:              Product,
	token.Asterisk:           Product,
	token.Pow:                Product,
	token.LBracket:           Index,
	token.Dot:                Call,
	token.LParen:             Call,
	token.ResolutionOperator: Call,
	token.RightArrow:         Call,
	token.LeftArrow:          Call,
	token.Pipe:               Call,
	token.Catch:              Call,
	token.Finally:            Call,
	token.Ident:              Call,
	token.Class:              Call,
	token.Assign:             Assign,
	token.PlusEq:             Assign,
	token.MinusEq:            Assign,
	token.OrEq:               Assign,
	token.Colon:              Assign,
}
