package vm

// RObject represents any non built-in class's instance.
type RObject struct {
	BaseObj
}

// ToString returns the object's name as the string format
func (ro *RObject) ToString(t *Thread) string {
	method := ro.FindMethod("string", false)
	methodObj, ok := method.(*MethodObject)
	if ok && t != nil {
		t.Stack.Push(ro)
		t.evalMethodCall(ro, methodObj, t.Stack.pointer, 0, nil, nil, t.GetSourceLine())
		result := t.Stack.Pop()
		return result.ToString(t)
	}
	return "#<" + ro.class.Name + ":instance >"
}

// Inspect delegates to ToString
func (ro *RObject) Inspect(t *Thread) string {
	var iv string

	for _, n := range ro.instanceVariables.names() {
		v, _ := ro.GetVariable(n)
		iv = iv + n + "=" + v.ToString(t) + " "
	}
	return "#<" + ro.class.Name + ":instance " + iv + ">"
}

// ToJSON just delegates to ToString
func (ro *RObject) ToJSON(t *Thread) string {
	method := ro.FindMethod("json", false)
	methodObj, ok := method.(*MethodObject)

	if ok && t != nil {
		t.Stack.Push(ro)
		t.evalMethodCall(ro, methodObj, t.Stack.pointer, 0, nil, nil, t.GetSourceLine())
		result := t.Stack.Pop()
		return result.ToString(t)
	}
	return ro.ToString(t)
}

// Value returns object's string format
func (ro *RObject) Value() interface{} {
	return ro.ToString(nil)
}

// EqualTo returns if the RObject is equal to another object
func (ro *RObject) EqualTo(with Object) bool {
	return ro == with
}
