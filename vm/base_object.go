package vm

// BaseObj ...
type BaseObj struct {
	class             *RClass
	instanceVariables Environment
}

// BaseObject returns a base obj with the given class
func BaseObject(c *RClass) BaseObj {
	return BaseObj{class: c}
}

// Class will return object's class
func (b *BaseObj) Class() *RClass {
	if b.class == nil {
		panic("Object doesn't have class.")
	}

	return b.class
}

// GetVariable gets an instance variable from the object
func (b *BaseObj) GetVariable(name string) (Object, bool) {
	if b.instanceVariables != nil {
		v, ok := b.instanceVariables[name]
		if ok {
			return v, true
		}
	}
	return NIL, false
}

// SetVariable sets an instance variable on the object
func (b *BaseObj) SetVariable(name string, value Object) Object {
	if b.instanceVariables == nil {
		b.instanceVariables = Environment{}
	}
	b.instanceVariables[name] = value
	return value
}

// Variables returns the current Environment for the object
func (b *BaseObj) Variables() Environment {
	if b.instanceVariables == nil {
		b.instanceVariables = Environment{}
	}
	return b.instanceVariables
}

// SetVariables sets the variables  in the supplied environment
func (b *BaseObj) SetVariables(e Environment) {
	b.instanceVariables = e
}

// FindMethod returns the method with the corresponding name
func (b *BaseObj) FindMethod(methodName string, super bool) (method Object) {
	class := b.class
	if super {
		class = class.superClass
	}
	return class.lookupMethod(methodName)
}

// FindLookup ...
func (b *BaseObj) FindLookup(searchAncestor bool) (method Object) {
	method = b.class.Methods[lookupMethod]
	if method == nil && searchAncestor {
		method = b.class.lookupMethod(lookupMethod)
	}
	return
}

// IsTruthy returns the boolean representation of the object
func (b *BaseObj) IsTruthy() bool {
	return true
}

// EqualTo returns true if the two objects are equivalent
func (b *BaseObj) EqualTo(with Object) bool {
	return b.Class().EqualTo(with.Class()) &&
		b.Variables().EqualTo(with.Variables())
}

// Inspect ...
func (b *BaseObj) Inspect(t *Thread) string {
	return b.ToString(t)
}

// ToJSON ...
func (b *BaseObj) ToJSON(t *Thread) string {
	return b.ToString(t)
}

// ToString ...
func (b *BaseObj) ToString(t *Thread) string {
	return ""
}

// Value ...
func (b *BaseObj) Value() interface{} {
	return b
}
