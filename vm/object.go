package vm

// Object represents all objects in Lito, including Array,
// Integer or even Method and Error.
type Object interface {
	Class() *RClass
	Value() interface{}
	FindMethod(string, bool) Object
	FindLookup(bool) Object
	ToString(t *Thread) string
	Inspect(t *Thread) string
	ToJSON(t *Thread) string
	GetVariable(string) (Object, bool)
	SetVariable(string, Object) Object
	Variables() Environment
	SetVariables(Environment)
	IsTruthy() bool
	EqualTo(Object) bool
}

const (
	lookupMethod  = "lookup!"
	receiveMethod = "receive"
	initMethod = "init"
)
