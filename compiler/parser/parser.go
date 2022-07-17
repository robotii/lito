package parser

import (
	"fmt"

	"github.com/robotii/lito/compiler/ast"
	"github.com/robotii/lito/compiler/lexer"
	"github.com/robotii/lito/compiler/parser/errors"
	"github.com/robotii/lito/compiler/parser/precedence"
	"github.com/robotii/lito/compiler/parser/states"
	"github.com/robotii/lito/compiler/token"
	"github.com/robotii/lito/fsm"
)

// Parser represents lexical analyzer struct
type Parser struct {
	Lexer lexer.Lexer
	error *errors.Error

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.Type]prefixParseFn
	infixParseFns  map[token.Type]infixParseFn

	// Determine if call expression should accept block argument,
	// currently only used when parsing while statement.
	// However, this is not a very good practice should change it in the future.
	acceptBlock bool
	fsm         *fsm.FSM
	Mode        Mode
}

// Mode determines the running mode. These are the enums for marking parser's mode, which decides whether it should pop unused values.
type Mode int

// These are the enums for marking parser's mode, which decides whether it should pop unused values.
const (
	NormalMode Mode = iota
	REPLMode
	TestMode
	CommandLineMode
)

// New creates a new parser and returns it
func New(l lexer.Lexer, mode Mode) *Parser {
	p := &Parser{
		Lexer:       l,
		Mode:        mode,
		acceptBlock: true,
	}

	p.fsm = fsm.New(
		states.Normal,
		fsm.States{
			{Name: states.ParsingFuncCall, From: []string{states.Normal}},
			{Name: states.ParsingMethodParam, From: []string{states.Normal, states.ParsingAssignment}},
			{Name: states.ParsingAssignment, From: []string{states.Normal, states.ParsingFuncCall}},
			{Name: states.Normal, From: []string{states.ParsingFuncCall, states.ParsingMethodParam, states.ParsingAssignment}},
		},
	)

	p.prefixParseFns = make(map[token.Type]prefixParseFn)
	p.registerPrefix(token.Ident, p.parseIdentifier)
	p.registerPrefix(token.Constant, p.parseConstant)
	p.registerPrefix(token.InstanceVariable, p.parseInstanceVariable)
	p.registerPrefix(token.Int, p.parseIntegerLiteral)
	p.registerPrefix(token.String, p.parseStringLiteral)
	p.registerPrefix(token.True, p.parseBooleanLiteral)
	p.registerPrefix(token.False, p.parseBooleanLiteral)
	p.registerPrefix(token.Nil, p.parseNilExpression)
	p.registerPrefix(token.Minus, p.parsePrefixExpression)
	p.registerPrefix(token.Plus, p.parsePrefixExpression)
	p.registerPrefix(token.Asterisk, p.parsePrefixExpression)
	p.registerPrefix(token.Bang, p.parsePrefixExpression)
	p.registerPrefix(token.Amp, p.parsePrefixExpression)
	p.registerPrefix(token.LParen, p.parseGroupedExpression)
	p.registerPrefix(token.If, p.parseIfExpression)
	p.registerPrefix(token.Switch, p.parseSwitchExpression)
	p.registerPrefix(token.Self, p.parseSelfExpression)
	p.registerPrefix(token.Super, p.parseSelfExpression)
	p.registerPrefix(token.LBracket, p.parseArrayExpression)
	p.registerPrefix(token.LBrace, p.parseHashExpression)
	p.registerPrefix(token.Semicolon, p.parseSemicolon)
	p.registerPrefix(token.Yield, p.parseYieldExpression)
	p.registerPrefix(token.GetBlock, p.parseGetBlockExpression)
	p.registerPrefix(token.HasBlock, p.parseHasBlockExpression)
	p.registerPrefix(token.RightArrow, p.parsePrefixExpression)
	p.registerPrefix(token.LeftArrow, p.parsePrefixExpression)

	p.infixParseFns = make(map[token.Type]infixParseFn)
	p.registerInfix(token.Plus, p.parseInfixExpression)
	p.registerInfix(token.PlusEq, p.parseAssignExpression)
	p.registerInfix(token.Minus, p.parseInfixExpression)
	p.registerInfix(token.MinusEq, p.parseAssignExpression)
	p.registerInfix(token.Modulo, p.parseInfixExpression)
	p.registerInfix(token.Slash, p.parseInfixExpression)
	p.registerInfix(token.Pow, p.parseInfixExpression)
	p.registerInfix(token.Eq, p.parseInfixExpression)
	p.registerInfix(token.NotEq, p.parseInfixExpression)
	p.registerInfix(token.IsSame, p.parseInfixExpression)
	p.registerInfix(token.IsNotSame, p.parseInfixExpression)
	p.registerInfix(token.Match, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.LTE, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.GTE, p.parseInfixExpression)
	p.registerInfix(token.And, p.parseInfixExpression)
	p.registerInfix(token.Or, p.parseInfixExpression)
	p.registerInfix(token.OrEq, p.parseAssignExpression)
	p.registerInfix(token.Comma, p.parseMultiVariables)
	p.registerInfix(token.ResolutionOperator, p.parseInfixExpression)
	p.registerInfix(token.Assign, p.parseAssignExpression)
	p.registerInfix(token.Range, p.parseRangeExpression)
	p.registerInfix(token.RangeExcl, p.parseRangeExpression)
	p.registerInfix(token.Dot, p.parseCallExpressionWithReceiver)
	p.registerInfix(token.LParen, p.parseCallExpressionWithoutReceiver)
	p.registerInfix(token.LBracket, p.parseIndexExpression)
	p.registerInfix(token.Colon, p.parseArgumentPairExpression)
	p.registerInfix(token.Asterisk, p.parseInfixExpression)
	p.registerInfix(token.RightArrow, p.parseInfixExpression)
	p.registerInfix(token.LeftArrow, p.parseInfixExpression)
	p.registerInfix(token.Pipe, p.parseInfixExpression)
	p.registerInfix(token.Catch, p.parseOperatorMethodCall)
	p.registerInfix(token.Finally, p.parseOperatorMethodCall)
	p.registerInfix(token.Ident, p.parseOperatorMethodCall)

	return p
}

// ParseProgram update program statements and return program
func (p *Parser) ParseProgram() (program *ast.Program, err *errors.Error) {

	defer func() {
		if recover() != nil {
			err = p.error
			if err == nil {
				msg := fmt.Sprintf("Internal error on token: %s. Line: %d", p.curToken.Literal, p.curToken.Line)
				err = errors.InitError(msg, errors.SyntaxError)
			}
		}
	}()

	p.error = nil
	// Read two tokens, so curToken and peekToken are both set.
	p.nextToken()
	p.nextToken()
	program = &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()

		if p.error != nil {
			return nil, p.error
		}

		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}

		p.nextToken()
	}

	// Set up last statement for testing, so we return the last value
	// that was returned from executing the last statement
	if p.Mode == TestMode || p.Mode == REPLMode {
		stmt := program.Statements[len(program.Statements)-1]
		expStmt, ok := stmt.(*ast.ExpressionStatement)
		if ok {
			expStmt.Expression.MarkAsExp()
		}
	}

	return program, nil
}

func (p *Parser) parseSemicolon() ast.Expression {
	return nil
}

func (p *Parser) peekPrecedence() int {
	return precedenceOf(p.peekToken.Type)
}

func (p *Parser) curPrecedence() int {
	return precedenceOf(p.curToken.Type)
}

func precedenceOf(t token.Type) int {
	if p, ok := precedence.LookupTable[t]; ok {
		return p
	}
	return precedence.Normal
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.Lexer.NextToken()
}

func (p *Parser) curTokenIs(t token.Type) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.Type) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.Type) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) registerPrefix(tokenType token.Type, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.Type, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) peekTokenAtSameLine() bool {
	return p.curToken.Line == p.peekToken.Line && p.peekToken.Type != token.EOF
}

func (p *Parser) peekError(t token.Type) {
	if p.peekToken.Type == token.EOF {
		msg := fmt.Sprintf("expected next token to be %s, got EOF(EOF) instead. Line: %d", t, p.peekToken.Line)
		p.error = errors.InitError(msg, errors.UnexpectedEOFError)
	} else {
		msg := fmt.Sprintf("expected next token to be %s, got %s(%s) instead. Line: %d", t, p.peekToken.Type, p.peekToken.Literal, p.peekToken.Line)
		p.error = errors.InitError(msg, errors.UnexpectedTokenError)
	}
}

func (p *Parser) noPrefixParseFnError(t token.Type) {
	msg := fmt.Sprintf("unexpected %s(%s) Line: %d", p.curToken.Literal, p.curToken.Type, p.curToken.Line+1)
	if t == token.RBrace {
		p.error = errors.InitError(msg, errors.UnexpectedEndError)
	} else if t == token.EOF && p.Mode == REPLMode {
		p.error = errors.InitError(msg, errors.UnexpectedEOFError)
	} else {
		p.error = errors.InitError(msg, errors.UnexpectedTokenError)
	}
}

func (p *Parser) callConstantError(t token.Type) {
	msg := fmt.Sprintf("cannot call %s with %s. Line: %d", t, p.peekToken.Type, p.peekToken.Line)
	p.error = errors.InitError(msg, errors.UnexpectedTokenError)
}

// IsNotDefMethodToken ensures correct naming in Def statement
func (p *Parser) IsNotDefMethodToken() bool {
	return p.curToken.Type != token.Ident && !(p.peekToken.Type == token.Dot && (p.curToken.Type == token.InstanceVariable || p.curToken.Type == token.Constant || p.curToken.Type == token.Self))
}

// Token type InstanceVariable and Constant will trigger IsNotParamsToken()
var invalidParams = map[token.Type]bool{
	token.InstanceVariable: true,
	token.Constant:         true,
}

// IsNotParamsToken ensures correct parameters which means it is not InstanceVariable
func (p *Parser) IsNotParamsToken() bool {
	_, ok := invalidParams[p.curToken.Type]
	return ok
}
