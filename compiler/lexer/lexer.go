package lexer

import (
	"github.com/robotii/lito/compiler/token"
	"github.com/robotii/lito/fsm"
)

// Lexer the interface between the parser and the lexer
type Lexer interface {
	NextToken() token.Token
}

// mLexer structure used to convert programs into a stream of tokens.
type mLexer struct {
	input        []rune   // the current input buffer being lexed
	position     int      // the index of the character we are currently lexing
	readPosition int      // the index of the next character to be read
	ch           rune     // the current character we are processing
	line         int      // the line number we are on
	fsm          *fsm.FSM // the finite state machine used to perform context-sensitive lexing
}

// FSM states
const (
	initial  = "initial"
	method   = "method"
	nosymbol = "nosymbol"
)

// New initialises a new lexer with input string
func New(input string) Lexer {
	l := &mLexer{input: []rune(input)}
	// Read the first character
	l.advance()

	l.fsm = fsm.New(
		initial,
		fsm.States{
			{Name: nosymbol, From: []string{initial}},
			{Name: method, From: []string{initial}},
			{Name: initial, From: []string{method, initial, nosymbol}},
		})
	return l
}

// NextToken lex and return the next token
func (l *mLexer) NextToken() token.Token {

	var tok token.Token
	l.resetNosymbol()
	l.skipWhitespace()

	switch l.ch {
	case '"', '\'':
		tok.Literal = l.readString(l.ch)
		tok.Type = token.String
		tok.Line = l.line
		return tok
	case '=':
		if l.peek() == '=' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				tok = token.CreateOperator("===", l.line)
			} else {
				tok = token.CreateOperator("==", l.line)
			}
		} else if l.peek() == '~' {
			l.advance()
			tok = token.CreateOperator("=~", l.line)
		} else {
			tok = token.CreateOperator("=", l.line)
		}
	case '-':
		if l.peek() == '=' {
			tok = token.CreateOperator("-=", l.line)
			l.advance()
			l.advance()
			return tok
		} else if l.peek() == '>' {
			l.advance()
			l.advance()
			tok = token.CreateOperator("->", l.line)
			return tok
		}
		tok = token.CreateOperator("-", l.line)
	case '!':
		if l.peek() == '=' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				tok = token.CreateOperator("!==", l.line)
			} else {
				tok = token.CreateOperator("!=", l.line)
			}
		} else {
			tok = token.CreateOperator("!", l.line)
		}
	case '/':
		tok = token.CreateOperator("/", l.line)
	case '*':
		if l.peek() == '*' {
			l.advance()
			tok = token.CreateOperator("**", l.line)
		} else {
			tok = token.CreateOperator("*", l.line)
		}
	case '<':
		if l.peek() == '=' {
			l.advance()
			tok = token.CreateOperator("<=", l.line)
		} else if l.peek() == '-' {
			l.advance()
			tok = token.CreateOperator("<-", l.line)
		} else {
			tok = token.CreateOperator("<", l.line)
		}
	case '>':
		if l.peek() == '=' {
			l.advance()
			tok = token.CreateOperator(">=", l.line)
		} else {
			tok = token.CreateOperator(">", l.line)
		}
	case ';', ',', '(', ')', '{', '}', '[', ']':
		tok = token.CreateSeparator(string(l.ch), l.line)
	case '+':
		if l.peek() == '=' {
			tok = token.CreateOperator("+=", l.line)
			l.advance()
			l.advance()
			return tok
		}
		tok = token.CreateOperator("+", l.line)
	case '.':
		if l.peek() == '.' {
			l.advance()
			if l.peek() == '.' {
				l.advance()
				tok = token.CreateOperator("...", l.line)
			} else {
				tok = token.CreateOperator("..", l.line)
			}
		} else {
			tok = token.CreateOperator(".", l.line)
			l.fsm.State(method)
		}
	case ':':
		if l.fsm.Is(nosymbol) {
			tok = token.CreateSeparator(":", l.line)
		} else {
			if l.peek() == ':' {
				l.advance()
				tok = token.CreateOperator("::", l.line)
			} else if isLetter(l.peek()) {
				return token.Create(token.String, l.readSymbol(), l.line)
			} else {
				tok = token.CreateSeparator(":", l.line)
			}
		}
	case '|':
		if l.peek() == '|' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				tok = token.CreateOperator("||=", l.line)
			} else {
				tok = token.CreateOperator("||", l.line)
			}
		} else if l.peek() == '>' {
			l.advance()
			tok = token.CreateOperator("|>", l.line)
		} else {
			tok = token.CreateSeparator("|", l.line)
		}
	case '%':
		tok = token.CreateOperator("%", l.line)
	case '#':
		return token.Create(token.Comment, l.readComment(), l.line)
	case '&':
		if l.peek() == '&' {
			l.advance()
			tok = token.CreateOperator("&&", l.line)
			break
		}
		tok = token.CreateOperator("&", l.line)
	case 0:
		tok = token.Create(token.EOF, "", l.line)
	default:
		if isLetter(l.ch) || l.ch == '`' {
			if 'A' <= l.ch && l.ch <= 'Z' {
				tok = token.Create(token.Constant, l.readConstant(), l.line)
				l.fsm.State(initial)
			} else {
				// Handle quoted identifier
				// TODO: Give this its own case block?
				if l.ch == '`' {
					tok.Literal = l.readQuotedIdentifier()
					// Make sure we return the correct type for quoted identifier
					if len(tok.Literal) == 0 {
						return token.Create(token.Illegal, "``", l.line)
					}
					if 'A' <= tok.Literal[0] && tok.Literal[0] <= 'Z' {
						tok.Type = token.Constant
						tok.Line = l.line
						l.fsm.State(initial)
						return tok
					}
					if isInstanceVariable(rune(tok.Literal[0])) {
						tok.Type = token.InstanceVariable
						tok.Line = l.line
						return tok
					}
				} else {
					tok.Literal = l.readIdentifier()
				}
				if l.fsm.Is(method) {
					if tok.Literal == "self" {
						tok.Type = token.LookupIdent(tok.Literal)
					} else {
						tok.Type = token.Ident
					}
					l.fsm.State(initial)
				} else if l.fsm.Is(initial) {
					tok.Type = token.LookupIdent(tok.Literal)
					if tok.Literal == "def" {
						l.fsm.State(method)
					}
				}
				tok.Line = l.line
			}
			if tok.Type == token.Ident {
				l.fsm.State(nosymbol)
			}
			return tok
		} else if isInstanceVariable(l.ch) {
			if isLetter(l.peek()) {
				return token.Create(token.InstanceVariable, l.readInstanceVariable(), l.line)
			}
			return token.Create(token.Illegal, string(l.ch), l.line)
		} else if isDigit(l.ch) {
			return token.Create(token.Int, l.readNumber(), l.line)
		}

		tok = token.Create(token.Illegal, string(l.ch), l.line)
	}

	l.advance()
	return tok
}

func (l *mLexer) skipWhitespace() {
	for isWhitespace(l.ch) {
		if l.ch == '\n' {
			l.line++
		}
		l.advance()
	}
}

func (l *mLexer) resetNosymbol() {
	if !l.fsm.Is(method) && l.ch != ':' {
		l.fsm.State(initial)
	}
}

func (l *mLexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.advance()
	}
	return string(l.input[position:l.position])
}

func (l *mLexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.advance()
	}

	if l.ch == '?' || l.ch == '!' {
		l.advance()
	}

	return string(l.input[position:l.position])
}

func (l *mLexer) readQuotedIdentifier() string {
	l.advance()
	p := l.position
	for l.ch != '`' && l.ch != '\n' && l.ch != 0 {
		l.advance()
	}
	result := l.input[p:l.position]
	// Skip past the terminating '`'
	l.advance()
	return string(result)
}

func (l *mLexer) readConstant() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.advance()
	}
	return string(l.input[position:l.position])
}

func (l *mLexer) readInstanceVariable() string {
	position := l.position
	for isLetter(l.ch) || isInstanceVariable(l.ch) || isDigit(l.ch) {
		l.advance()
	}
	return string(l.input[position:l.position])
}

func (l *mLexer) readString(ch rune) string {
	l.advance()

	// Empty strings case such as "" or ''
	if l.ch == ch {
		l.advance()
		return ""
	}

	result := ""

	for {
		if isEscapeChar(l.ch) {
			l.advance()
			r, ok := l.escapeSequence()
			if ok {
				result += string(r)
			}
		} else {
			result += string(l.ch)
		}
		l.advance()
		if l.ch == ch || l.peek() == 0 {
			break
		}
	}

	l.advance() // move to string's latter quote
	return result
}

func (l *mLexer) readSymbol() string {
	// Consume the ':' character
	l.advance()
	return l.readIdentifier()
}

func (l *mLexer) readComment() string {
	p := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.advance()
	}

	return string(l.input[p:l.position])
}

func (l *mLexer) advance() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

func (l *mLexer) peek() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isInstanceVariable(ch rune) bool {
	return ch == '@'
}

func isEscapeChar(ch rune) bool {
	return ch == '\\'
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

// escapeSequence roughly follows Go semantics for reading escape sequences
func (l *mLexer) escapeSequence() (rune, bool) {
	c := l.ch
	switch c {
	case 'a':
		return '\a', true

	case 'b':
		return '\b', true

	case 'f':
		return '\f', true

	case 'n':
		return '\n', true

	case 'r':
		return '\r', true

	case 't':
		return '\t', true

	case 'v':
		return '\v', true

	case 'x', 'u', 'U':
		n := 0
		switch c {
		case 'x':
			n = 2
		case 'u':
			n = 4
		case 'U':
			n = 8
		}
		var v rune
		for i := 0; i < n; i++ {
			l.advance()
			ch := l.ch
			x, ok := fromhex(ch)
			if !ok {
				return 0, false
			}
			v = v<<4 | x
		}
		return v, true

	case '0', '1', '2', '3', '4', '5', '6', '7':
		v := c - '0'
		for i := 0; i < 2; i++ {
			l.advance()
			ch := l.ch
			x := ch - '0'
			if x < 0 || x > 7 {
				return 0, false
			}
			v = (v << 3) | x
		}
		return v, true

	case '\\':
		return '\\', true

	case '\'', '"':
		return c, true

	default:
		// Default to the character if it is not an escape sequence
		return c, true
	}
}

func fromhex(r rune) (rune, bool) {
	switch r {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return r - '0', true
	case 'a', 'b', 'c', 'd', 'e', 'f':
		return r - 'a' + 10, true
	case 'A', 'B', 'C', 'D', 'E', 'F':
		return r - 'A' + 10, true
	default:
		return 0, false
	}
}
