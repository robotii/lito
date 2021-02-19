package token

// Type is used to determine token type
type Type string

// Token is structure for identifying input stream of characters
type Token struct {
	Type    Type
	Literal string
	Line    int
}

// Literals
const (
	Illegal = "ILLEGAL"
	EOF     = "EOF"

	Constant         = "CONSTANT"
	Ident            = "IDENT"
	InstanceVariable = "INSTANCE_VAR"
	Int              = "INT"
	Float            = "FLOAT"
	String           = "STRING"
	Comment          = "COMMENT"

	Assign   = "="
	Plus     = "+"
	PlusEq   = "+="
	Minus    = "-"
	MinusEq  = "-="
	Bang     = "!"
	Asterisk = "*"
	Pow      = "**"
	Slash    = "/"
	Dot      = "."
	And      = "&&"
	Or       = "||"
	OrEq     = "||="
	Modulo   = "%"

	Match = "=~"
	LT    = "<"
	LTE   = "<="
	GT    = ">"
	GTE   = ">="
	COMP  = "<=>" // TODO: Remove

	Comma     = ","
	Semicolon = ";"
	Colon     = ":"
	Bar       = "|"
	Amp       = "&"

	LParen   = "("
	RParen   = ")"
	LBrace   = "{"
	RBrace   = "}"
	LBracket = "["
	RBracket = "]"

	Eq        = "=="
	NotEq     = "!="
	IsSame    = "==="
	IsNotSame = "!=="
	Range     = ".."
	RangeExcl = "..."

	True     = "TRUE"
	False    = "FALSE"
	Nil      = "NIL"
	If       = "IF"
	ElsIf    = "ELSIF"
	Else     = "ELSE"
	Default  = "DEFAULT"
	Switch   = "SWITCH"
	Case     = "CASE"
	Return   = "RETURN"
	Continue = "CONTINUE"
	Break    = "BREAK"
	Def      = "DEF"
	Self     = "SELF"
	Super    = "SUPER"
	While    = "WHILE"
	Yield    = "YIELD"
	GetBlock = "BLOCK"
	HasBlock = "HASBLOCK"
	Class    = "CLASS"
	Module   = "MODULE"
	Catch    = "CATCH"
	Finally  = "FINALLY"

	ResolutionOperator = "::"

	RightArrow = "->"
	LeftArrow  = "<-"
	Pipe       = "|>"
)

var keywords = map[string]Type{
	"def":      Def,
	"true":     True,
	"false":    False,
	"nil":      Nil,
	"if":       If,
	"elsif":    ElsIf,
	"else":     Else,
	"switch":   Switch,
	"case":     Case,
	"default":  Default,
	"return":   Return,
	"self":     Self,
	"super":    Super,
	"while":    While,
	"yield":    Yield,
	"continue": Continue,
	"class":    Class,
	"module":   Module,
	"break":    Break,
	"block!":   GetBlock,
	"block?":   HasBlock,
	"catch":    Catch,
	"finally":  Finally,
}

var operators = map[string]Type{
	"=":   Assign,
	"+":   Plus,
	"+=":  PlusEq,
	"-":   Minus,
	"-=":  MinusEq,
	"!":   Bang,
	"*":   Asterisk,
	"**":  Pow,
	"/":   Slash,
	".":   Dot,
	"&&":  And,
	"||":  Or,
	"||=": OrEq,
	"%":   Modulo,
	"&":   Amp,

	"=~":  Match,
	"<":   LT,
	"<=":  LTE,
	">":   GT,
	">=":  GTE,
	"<=>": COMP, // TODO: Remove

	"==":  Eq,
	"!=":  NotEq,
	"===": IsSame,
	"!==": IsNotSame,
	"..":  Range,
	"...": RangeExcl,

	"::": ResolutionOperator,
	"->": RightArrow,
	"<-": LeftArrow,
	"|>": Pipe,
}

var separators = map[string]Type{
	",": Comma,
	";": Semicolon,
	":": Colon,
	"|": Bar,

	"(": LParen,
	")": RParen,
	"{": LBrace,
	"}": RBrace,
	"[": LBracket,
	"]": RBracket,
}

// LookupIdent is used for keyword identification
func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return Ident
}

func getOperatorType(literal string) Type {
	if t, ok := operators[literal]; ok {
		return t
	}
	return Ident
}

func getSeparatorType(literal string) Type {
	if t, ok := separators[literal]; ok {
		return t
	}
	return Ident
}

// CreateOperator - Factory method for creating operator types token from literal string
func CreateOperator(literal string, line int) Token {
	return Token{Type: getOperatorType(literal), Literal: literal, Line: line}
}

// CreateSeparator - Factory method for creating separator types token from literal string
func CreateSeparator(literal string, line int) Token {
	return Token{Type: getSeparatorType(literal), Literal: literal, Line: line}
}

// Create will create a token
func Create(t Type, literal string, line int) Token {
	return Token{Type: t, Literal: literal, Line: line}
}
