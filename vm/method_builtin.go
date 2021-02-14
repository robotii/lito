package vm

// NoSuchMethod convenience function for when the method does not exist
func NoSuchMethod(name string) Method {
	return func(receiver Object, t *Thread, args []Object) Object {
		return t.vm.InitNoMethodError(t, name, receiver)
	}
}

// BuiltinMethodObject represents methods defined in go.
type BuiltinMethodObject struct {
	BaseObj
	Name      string
	Fn        Method
	Primitive bool
}

// Method is a callable function
type Method = func(receiver Object, t *Thread, args []Object) Object

// ExternalBuiltinMethod is a function that builds a BuiltinMethodObject from an external function
func ExternalBuiltinMethod(name string, m Method) *BuiltinMethodObject {
	return &BuiltinMethodObject{
		Name: name,
		Fn:   m,
	}
}

// ToString returns the object's name as the string format
func (bim *BuiltinMethodObject) ToString(t *Thread) string {
	return "<BuiltinMethod: " + bim.Name + ">"
}

// Inspect delegates to ToString
func (bim *BuiltinMethodObject) Inspect(t *Thread) string {
	return bim.ToString(t)
}

// ToJSON just delegates to `ToString`
func (bim *BuiltinMethodObject) ToJSON(t *Thread) string {
	return bim.ToString(t)
}

// Value returns builtin method object's function
func (bim *BuiltinMethodObject) Value() interface{} {
	return bim.Fn
}
