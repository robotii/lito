package vm

import (
	"fmt"
	"path"
	"sync"
	"time"

	"sort"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// RClass represents normal (not built in) class object
type RClass struct {
	BaseObj
	// Name is the class's name
	Name string
	// Methods contains its instances' methods
	Methods Environment
	// pseudoSuperClass points to the class it inherits
	pseudoSuperClass *RClass
	// metaClass stores the class methods
	metaClass *RClass
	// This is the class where we should looking for a method.
	// It can be normal class, meta class or a module.
	superClass *RClass
	// Class points to this class's class, which should be ClassClass
	isModule       bool
	constants      map[string]*Pointer
	scope          *RClass
	inheritsLookup bool
}

// ClassLoader can be registered with a vm so that it can load this library at vm creation
type ClassLoader = func(vm *VM) error

var externalClasses = map[string][]ClassLoader{}
var externalClassLock sync.Mutex

// RegisterExternalClass will add the given class to the global registery of available classes
func RegisterExternalClass(path string, c ...ClassLoader) {
	externalClassLock.Lock()
	externalClasses[path] = c
	externalClassLock.Unlock()
}

func buildMethods(m map[string]Method) []*BuiltinMethodObject {
	out := make([]*BuiltinMethodObject, 0, len(m))
	for k, v := range m {
		out = append(out, ExternalBuiltinMethod(k, v))
	}
	return out
}

// ExternalClass helps define external go classes
func ExternalClass(name, path string, classMethods, instanceMethods map[string]Method) ClassLoader {
	return func(vm *VM) error {
		vm.objectClass.SetClassConstant(
			vm.InitClass(name).
				ClassMethods(buildMethods(classMethods)).
				InstanceMethods(buildMethods(instanceMethods)))

		if path == "" {
			return nil
		}
		return vm.newThread().loadLibrary(path)
	}
}

// Class's class methods
var classCommonClassMethods = []*BuiltinMethodObject{
	{
		Name: "new",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			class, ok := receiver.(*RClass)
			if !ok {
				return t.vm.InitNoMethodError(t, "new", receiver)
			}
			return class.initInstance()
		},
	},
}

var moduleCommonClassMethods = []*BuiltinMethodObject{
	{
		Name: "ancestors",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			c, ok := receiver.(*RClass)
			if !ok {
				return t.vm.InitNoMethodError(t, "ancestors", receiver)
			}

			a := c.ancestors()
			ancestors := make([]Object, len(a))
			for i := range a {
				ancestors[i] = a[i]
			}
			return InitArrayObject(ancestors)
		},
	},
	{
		Name: ">",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			c, ok := receiver.(*RClass)
			if !ok {
				return t.vm.InitNoMethodError(t, ">", receiver)
			}

			if module, ok := args[0].(*RClass); ok {
				if c == module {
					return FALSE
				}
				if module.alreadyInherit(c) {
					return TRUE
				}
				if c.alreadyInherit(module) {
					return FALSE
				}
				return NIL
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "a module", args[0].Class().Name)
		},
	},
	{
		Name: ">=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			c, ok := receiver.(*RClass)
			if !ok {
				return t.vm.InitNoMethodError(t, ">=", receiver)
			}

			if module, ok := args[0].(*RClass); ok {
				if c == module {
					return TRUE
				}
				if module.alreadyInherit(c) {
					return TRUE
				}
				if c.alreadyInherit(module) {
					return FALSE
				}
				return NIL
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "a module", args[0].Class().Name)
		},
	},
	{
		Name: "<",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			c, ok := receiver.(*RClass)
			if !ok {
				return t.vm.InitNoMethodError(t, "<", receiver)
			}

			module, ok := args[0].(*RClass)
			if ok {
				if c == module {
					return FALSE
				}
				if module.alreadyInherit(c) {
					return FALSE
				}
				if c.alreadyInherit(module) {
					return TRUE
				}
				return NIL
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "a module", args[0].Class().Name)
		},
	},
	{
		Name: "<=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			c, ok := receiver.(*RClass)
			if !ok {
				return t.vm.InitNoMethodError(t, "<=", receiver)
			}

			module, ok := args[0].(*RClass)
			if ok {
				if c == module {
					return TRUE
				}
				if module.alreadyInherit(c) {
					return FALSE
				}
				if c.alreadyInherit(module) {
					return TRUE
				}
				return NIL
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "a module", args[0].Class().Name)
		},
	},
	{
		Name: "property",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) == 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentMore, 0, len(args))
			}
			// TODO: Check for strings
			receiver.(*RClass).addProperty(args)
			return receiver
		},
	},
	{
		Name: "get",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) == 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentMore, 1, len(args))
			}
			// TODO: Check for strings
			receiver.(*RClass).addGetter(args)
			return receiver
		},
	},
	{
		Name: "set",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) == 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentMore, 1, len(args))
			}
			// TODO: Check for strings
			receiver.(*RClass).addSetter(args)
			return receiver
		},
	},
	{
		Name: "constants",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			var constantNames []string
			var objs []Object

			for n := range receiver.(*RClass).constants {
				constantNames = append(constantNames, n)
			}
			sort.Strings(constantNames)

			for _, cn := range constantNames {
				objs = append(objs, StringObject(cn))
			}

			return InitArrayObject(objs)
		},
	},
	{
		Name: "extend",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			var class *RClass
			module, ok := args[0].(*RClass)
			if !ok || !module.isModule {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "a module", args[0].Class().Name)
			}

			class = receiver.(*RClass).MetaClass()

			if class.alreadyInherit(module) {
				return class
			}

			// Make a copy of the module
			module = &*module
			module.superClass = class.superClass
			class.superClass = module

			return class
		},
	},
	{
		Name: "include",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			var class *RClass
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			module, ok := args[0].(*RClass)
			if !ok || !module.isModule {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "a module", args[0].Class().Name)
			}

			switch r := receiver.(type) {
			case *RClass:
				class = r
			default:
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "a class", r.Class().Name)
			}

			if class.alreadyInherit(module) {
				return class
			}

			// Make a copy of the module
			module = &*module
			module.superClass = class.superClass
			class.superClass = module

			return class
		},
	},
	{
		Name: "inherits_lookup!",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			var class *RClass

			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			switch r := receiver.(type) {
			case *RClass:
				class = r
			default:
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "a class", r.Class().Name)
			}

			if class != nil {
				class.inheritsLookup = true
			}
			return receiver
		},
	},
	{
		Name: "name",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			n, ok := receiver.(*RClass)
			if !ok {
				return t.vm.InitNoMethodError(t, "name", receiver)
			}
			return StringObject(n.Name)
		},
	},
	{
		Name: "respond_to?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			arg, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, arg.Class().Name)
			}

			return BooleanObject(receiver.FindMethod(string(arg), false) != nil)
		},
	},
	{
		Name: "superclass",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			c, ok := receiver.(*RClass)

			if !ok {
				return t.vm.InitNoMethodError(t, "superclass", receiver)
			}

			superClass := c.returnSuperClass()

			if superClass == nil {
				return NIL
			}

			return superClass
		},
	},
}

var classCommonInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "==",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if receiver == args[0] {
				return TRUE
			}
			return BooleanObject(receiver.EqualTo(args[0]))
		},
	},
	{
		Name: "===",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if receiver == args[0] {
				return TRUE
			}
			switch receiver.(type) {
			case IntegerObject, FloatObject, StringObject:
				return BooleanObject(receiver.EqualTo(args[0]))
			}
			return FALSE
		},
	},
	{
		Name: "!=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if receiver == args[0] {
				return FALSE
			}
			return BooleanObject(!receiver.EqualTo(args[0]))
		},
	},
	{
		Name: "!==",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if receiver == args[0] {
				return FALSE
			}
			switch receiver.(type) {
			case IntegerObject, FloatObject, StringObject:
				return BooleanObject(!receiver.EqualTo(args[0]))
			}
			return TRUE
		},
	},
	{
		Name: "!",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			rightValue, ok := receiver.(BooleanObject)
			return BooleanObject(ok && !bool(rightValue))
		},
		Primitive: true,
	},
	{
		Name: "class",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return receiver.Class()
		},
	},
	{
		Name: "dup",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			switch receiver.(type) {
			case *RObject:
				newObj := receiver.Class().initInstance()
				newObj.SetVariables(receiver.Variables().copy())
				return newObj
			default:
				return receiver
			}
		},
	},
	{
		Name: "instance_of?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			rClass, ok := args[0].(*RClass)
			if ok {
				return BooleanObject(receiver.Class().Name == rClass.Name)
			}
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.ClassClass, args[0].Class().Name)
		},
		Primitive: true,
	},
	{
		Name:      "is_a?",
		Fn:        classIsA,
		Primitive: true,
	},
	{
		Name: "inherits_lookup?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return BooleanObject(receiver.Class().inheritsLookup)
		},
		Primitive: true,
	},
	{
		// TODO: Can we combine with "tap"?
		Name: "instance_eval",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			blockFrame := t.GetBlock()
			aLen := len(args)
			switch aLen {
			case 0:
			case 1:
				if args[0].Class().Name == classes.BlockClass {
					blockObj := args[0].(*BlockObject)
					blockFrame = blockObj.asCallFrame(t)
				} else {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.BlockClass, args[0].Class().Name)
				}
			default:
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 1, aLen)
			}

			if blockFrame == nil || blockFrame.IsEmpty() {
				return receiver
			}

			blockFrame.self = receiver
			return t.Yield(blockFrame)
		},
	},
	{
		Name: "vget",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			arg, isStr := args[0].(StringObject)
			if !isStr {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			obj, ok := receiver.GetVariable(string(arg))
			if !ok {
				return NIL
			}

			return obj
		},
		Primitive: true,
	},
	{
		Name: "vset",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 2, len(args))
			}

			argName, isStr := args[0].(StringObject)
			if !isStr {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			receiver.SetVariable(string(argName), args[1])
			return args[1]
		},
		Primitive: true,
	},
	{
		Name: "methods",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			var klasses []*RClass
			if class, ok := receiver.(*RClass); ok {
				if class.MetaClass() != nil {
					klasses = append(klasses, class.MetaClass())
				}
			}
			methods := getMethods(append(klasses, receiver.Class().ancestors()...))
			return InitArrayObject(methods)
		},
		Primitive: true,
	},
	{
		Name: "own_methods",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			klasses := []*RClass{receiver.Class()}
			if class, ok := receiver.(*RClass); ok {
				if class.MetaClass() != nil {
					klasses = []*RClass{class.MetaClass(), receiver.Class()}
				}
			}
			methods := getMethods(klasses)
			return InitArrayObject(methods)
		},
		Primitive: true,
	},
	{
		Name: "nil?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return FALSE
		},
		Primitive: true,
	},
	{
		Name: "print",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			for _, arg := range args {
				fmt.Print(arg.ToString(t))
			}
			return NIL
		},
	},
	{
		Name: "println",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			for _, arg := range args {
				fmt.Print(arg.ToString(t))
			}
			fmt.Println()
			return NIL
		},
	},
	{
		// TODO: Fix error function to be more sane.
		Name: "raise",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			aLen := len(args)
			switch aLen {
			case 0:
				return t.vm.InitErrorObject(t, errors.Error, "")
			case 1:
				// If we supplied an error, we simply return it to throw the error
				if args[0].Class().isA(t.vm.errorClass) {
					// If we get passed an error, let's raise it!
					if err, ok := args[0].(*Error); ok {
						err.Ignore = false
						err.Raised = true
						err.storeStackTraces(t)
						err.storedTraces = true
						return err
					}
					return args[0]
				}
				return t.vm.InitErrorObject(t, errors.Error, "'%s'", args[0].ToString(t))
			case 2:
				errorClass, ok := args[0].(*RClass)
				if !ok {
					return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongArgumentTypeFormatNum, 2, "a class", args[0].Class().Name)
				}
				return t.vm.InitErrorObject(t, errorClass.Name, "'%s'", args[1].ToString(t))
			}
			return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentLess, 2, aLen)
		},
	},
	{
		Name: "catch",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}
			if len(args) == 0 {
				return t.Yield(blockFrame, receiver)
			}
			for _, catchtype := range args {
				if tClass, ok := catchtype.(*RClass); ok {
					if receiver.Class().isA(tClass) {
						// Make sure the error is not raised, if it is already
						if e, ok := receiver.(*Error); ok {
							e.Raised = false
							e.Ignore = false
						}
						return t.Yield(blockFrame, receiver)
					}
				}
			}
			return receiver
		},
	},
	{
		Name: "try",
		Fn: func(receiver Object, t *Thread, args []Object) (o Object) {
			// Turn off default panic behaviour for any errors that are raised
			// Any errors by the runtime will not be caught
			defer func() {
				switch err := recover().(type) {
				case error:
					// This is not an error we can deal with
					panic(err)
				case *Error:
					// Return the error
					err.Ignore = true
					o = err
				}
			}()
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				o = t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			} else {
				o = t.Yield(blockFrame, receiver)
			}
			return
		},
	},
	{
		// TODO: This is very similar to tap, maybe we can consolidate?
		Name: "finally",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			t.Yield(blockFrame, receiver)
			// Always return receiver
			return receiver
		},
	},
	{
		Name: "respond_to?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			arg, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, arg.Class().Name)
			}
			return BooleanObject(receiver.FindMethod(string(arg), false) != nil)
		},
	},
	{
		Name: "current_file",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return StringObject(t.vm.CurrentFilePath())
		},
		Primitive: true,
	},
	{
		Name: "require",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			switch args[0].(type) {
			case StringObject:
				libName := string(args[0].(StringObject))
				// TODO: Find a better way of loading in local dependencies
				if libName[0] == '.' {
					filePath := path.Join(path.Dir(t.vm.CurrentFilePath()), libName) + "." + FileExt

					if t.execFile(filePath) != nil {
						return t.vm.InitErrorObject(t, errors.IOError, errors.CantLoadFile, string(args[0].(StringObject)))
					}
				} else {
					initFunc, ok := standardLibraries[libName]

					if !ok {
						externalClassLock.Lock()
						loaders, ok := externalClasses[libName]
						externalClassLock.Unlock()
						if !ok {
							err := t.loadLibrary(libName + "." + FileExt)
							if err != nil {
								return t.vm.InitErrorObject(t, errors.IOError, errors.CantLoadFile, libName)
							}
						}
						initFunc = func(vm *VM) {
							for _, l := range loaders {
								_ = l(vm)
							}
						}
					}

					initFunc(t.vm)
				}

				return TRUE
			default:
				return t.vm.InitErrorObject(t, errors.TypeError, errors.CantRequireNonString, args[0].(Object).Class().Name)
			}
		},
	},
	{
		Name: "send",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) == 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentMore, 1, 0)
			}

			name, ok := args[0].(StringObject)
			if ok {
				t.sendMethod(string(name), len(args)-1, t.GetBlock())
				return t.Stack.top()
			}

			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
		},
	},
	{
		Name: "metaclass",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if class, ok := receiver.(*RClass); ok {
				if class.MetaClass() == nil {
					metaClass := createRClass(t.vm, fmt.Sprintf("#<Class:#<%s:metaclass>>", receiver.Class().Name))
					class.SetMetaClass(metaClass)
					return metaClass
				}
				return class.MetaClass()
			}
			return NIL
		},
	},
	{
		Name: "sleep",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			switch sleepTime := args[0].(type) {
			case IntegerObject:
				seconds := int(sleepTime)
				time.Sleep(time.Duration(seconds) * time.Second)
			case FloatObject:
				nanoseconds := int64(float64(sleepTime) * float64(time.Second/time.Nanosecond))
				time.Sleep(time.Duration(nanoseconds) * time.Nanosecond)
			default:
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Numeric", args[0].Class().Name)
			}
			return args[0]
		},
	},
	{
		Name: "tap",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			if blockFrame := t.GetBlock(); blockFrame != nil && !blockFrame.IsEmpty() {
				// Build a new callframe with the self set to the receiver
				cf := newNormalCallFrame(blockFrame.instructionSet, blockFrame.instructionSet.Filename, blockFrame.instructionSet.SourceMap[0])
				cf.ep = blockFrame.ep
				cf.self = receiver
				cf.isBlock = true
				t.Yield(cf, receiver)
			}

			// We always return the receiver
			return receiver
		},
	},
	{
		Name: "go",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			// TODO: How to handle errors in other threads?
			go func() {
				defer func() { recover() }()
				t.vm.newThread().Yield(blockFrame, args...)
			}()
			return NIL
		},
	},
	{
		Name: "string",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return StringObject(receiver.ToString(t))
		},
	},
	{
		Name: "inspect",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return StringObject(receiver.Inspect(t))
		},
	},
	{
		Name: "<-",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			ro := args[0]
			if c, ok := ro.(*ChannelObject); ok {
				return c.receive(t)
			}
			methodObj, ok := ro.FindMethod(receiveMethod, false).(*MethodObject)
			if ok {
				t.Stack.Push(ro)
				t.evalMethodCall(ro, methodObj, t.Stack.pointer, 0, nil, nil, t.GetSourceLine())
				return t.Stack.Pop()
			}
			return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongArgumentTypeFormat, classes.ChannelClass, args[0].Class().Name)
		},
	},
}

// InitClass is a common function for vm, which initialises and returns
// a class instance with given class name.
func (vm *VM) InitClass(name string) *RClass {
	class := createRClass(vm, name)
	metaClass := createRClass(vm, fmt.Sprintf("#<Class:%s>", name))
	class.metaClass = metaClass
	return class.inherits(vm.objectClass)
}

// InitModule creates a new module with the given name
func (vm *VM) InitModule(name string) *RClass {
	moduleClass := vm.TopLevelClass(classes.ModuleClass)
	module := createRClass(vm, name)
	module.class = moduleClass
	module.isModule = true
	metaClass := createRClass(vm, fmt.Sprintf("#<Class:%s>", name))
	metaClass.superClass = moduleClass
	metaClass.pseudoSuperClass = moduleClass
	module.metaClass = metaClass

	return module
}

func createRClass(vm *VM, className string) *RClass {
	return &RClass{
		Name:             className,
		Methods:          Environment{},
		pseudoSuperClass: vm.objectClass,
		superClass:       vm.objectClass,
		constants:        make(map[string]*Pointer),
		BaseObj:          BaseObj{class: vm.TopLevelClass(classes.ClassClass)},
	}
}

func initModuleClass(classClass *RClass) *RClass {
	moduleClass := &RClass{
		Name:      classes.ModuleClass,
		Methods:   Environment{},
		constants: make(map[string]*Pointer),
		BaseObj:   BaseObj{},
	}

	moduleMetaClass := &RClass{
		Name:      "#<Class:Module>",
		Methods:   Environment{},
		constants: make(map[string]*Pointer),
		BaseObj:   BaseObj{class: classClass},
	}

	classClass.superClass = moduleClass
	classClass.pseudoSuperClass = moduleClass

	moduleClass.class = classClass
	moduleClass.metaClass = moduleMetaClass

	return moduleClass.ClassMethods(moduleCommonClassMethods)
}

func initClassClass() *RClass {
	classClass := &RClass{
		Name:      classes.ClassClass,
		Methods:   Environment{},
		constants: make(map[string]*Pointer),
		BaseObj:   BaseObj{},
	}

	classMetaClass := &RClass{
		Name:      "#<Class:Class>",
		Methods:   Environment{},
		constants: make(map[string]*Pointer),
		BaseObj:   BaseObj{class: classClass},
	}

	classClass.class = classClass
	classClass.metaClass = classMetaClass

	return classClass.ClassMethods(classCommonClassMethods)
}

func initObjectClass(c *RClass) *RClass {
	objectClass := &RClass{
		Name:      classes.ObjectClass,
		Methods:   Environment{},
		constants: make(map[string]*Pointer),
		BaseObj:   BaseObj{class: c},
	}

	metaClass := &RClass{
		Name:       "#<Class:Object>",
		Methods:    Environment{},
		constants:  make(map[string]*Pointer),
		BaseObj:    BaseObj{class: c},
		superClass: c,
	}

	objectClass.metaClass = metaClass
	objectClass.superClass = objectClass
	objectClass.pseudoSuperClass = objectClass
	c.superClass.inherits(objectClass)

	return objectClass.ClassMethods(classCommonInstanceMethods).
		InstanceMethods(classCommonInstanceMethods)
}

// ToString returns the object's name as the string format
func (c *RClass) ToString(t *Thread) string {
	return c.Name
}

// Inspect delegates to ToString
func (c *RClass) Inspect(t *Thread) string {
	return c.ToString(t)
}

// ToJSON just delegates to `ToString`
func (c *RClass) ToJSON(t *Thread) string {
	return c.ToString(t)
}

// Value returns class itself
func (c *RClass) Value() interface{} {
	return c
}

// MetaClass returns the metaclass of the given class
func (c *RClass) MetaClass() *RClass {
	return c.metaClass
}

// SetMetaClass sets object's metaclass
func (c *RClass) SetMetaClass(m *RClass) {
	c.metaClass = m
}

// FindLookup ...
func (c *RClass) FindLookup(searchAncestor bool) (method Object) {
	metaClass := c.MetaClass()
	if metaClass != nil {
		method, _ = metaClass.Methods[lookupMethod]
	}
	if method == nil {
		method, _ = c.Class().Methods[lookupMethod]
	}
	if method == nil && searchAncestor {
		method = c.FindMethod(lookupMethod, false)
	}

	return
}

// FindMethod ...
func (c *RClass) FindMethod(methodName string, super bool) (method Object) {
	metaClass := c.metaClass
	class := c.class

	if super {
		class = class.superClass
		if metaClass != nil {
			metaClass = metaClass.superClass
		}
	}
	if metaClass != nil {
		method = metaClass.lookupMethod(methodName)
	}

	if method == nil {
		method = class.lookupMethod(methodName)
	}

	return
}

func (c *RClass) inherits(sc *RClass) *RClass {
	c.superClass = sc
	c.pseudoSuperClass = sc
	c.metaClass.superClass = sc.metaClass
	c.metaClass.pseudoSuperClass = sc.metaClass
	return c
}

// InstanceMethods adds the instance methods to the class
func (c *RClass) InstanceMethods(methodList []*BuiltinMethodObject) *RClass {
	for _, m := range methodList {
		c.Methods[m.Name] = m
	}
	return c
}

// ClassMethods adds the class methods to the class's metaclass
func (c *RClass) ClassMethods(methodList []*BuiltinMethodObject) *RClass {
	for _, m := range methodList {
		c.metaClass.Methods[m.Name] = m
		c.Methods[m.Name] = m
	}
	return c
}

func (c *RClass) lookupMethod(methodName string) Object {
	method, ok := c.Methods[methodName]
	if !ok {
		if c.superClass != nil && c.superClass != c {
			return c.superClass.lookupMethod(methodName)
		}

		return nil
	}

	return method
}

func (c *RClass) lookupConstantInCurrentScope(constName string) *Pointer {
	constant, ok := c.constants[constName]
	if !ok {
		return nil
	}
	return constant
}

func (c *RClass) lookupConstantUnderCurrentScope(constName string) *Pointer {
	constant, ok := c.constants[constName]
	if ok {
		return constant
	}
	if c.scope != nil {
		return c.scope.lookupConstantUnderCurrentScope(constName)
	}
	return nil
}

func (c *RClass) lookupConstantUnderAllScope(constName string) *Pointer {
	constant, ok := c.constants[constName]
	if ok {
		return constant
	}
	if c.scope != nil {
		return c.scope.lookupConstantUnderCurrentScope(constName)
	}
	// Finding constant in superclass means it's out of the scope
	if c.superClass != nil && c.Name != classes.ObjectClass {
		constant, _ = c.constants[constName]
		return constant
	}
	return nil
}

// SetClassConstant adds a class to the class's constants.
// Name is take from the class name of the supplied parameter.
func (c *RClass) SetClassConstant(constant *RClass) {
	c.constants[constant.Name] = &Pointer{Target: constant}
}

// SetConstant adds the constant to the class's constants, with the given name.
func (c *RClass) SetConstant(name string, constant Object) *RClass {
	c.constants[name] = &Pointer{Target: constant}
	return c
}

func (c *RClass) getClassConstant(constName string) (class *RClass) {
	t := c.constants[constName].Target
	class, ok := t.(*RClass)
	if ok {
		return
	}

	panic(constName + " is not a class.")
}

func (c *RClass) alreadyInherit(constant *RClass) bool {
	if c.superClass == constant {
		return true
	}

	if c.superClass.Name == classes.ObjectClass {
		return false
	}

	return c.superClass.alreadyInherit(constant)
}

func (c *RClass) returnSuperClass() *RClass {
	return c.pseudoSuperClass
}

func (c *RClass) initInstance() *RObject {
	return &RObject{BaseObj: BaseObj{class: c}}
}

func (c *RClass) addSetter(args []Object) {
	c.generateMethod(args, "=", generateSetMethod)
}

func (c *RClass) addGetter(args []Object) {
	c.generateMethod(args, "", generateGetMethod)
}

func (c *RClass) generateMethod(args []Object, suffix string, generate func(string) *BuiltinMethodObject) {
	for _, attr := range args {
		if attrName, ok := attr.(StringObject); ok {
			c.Methods[string(attrName)+suffix] = generate(string(attrName))
		}
	}
}

func (c *RClass) addProperty(args []Object) {
	c.addGetter(args)
	c.addSetter(args)
}

func (c *RClass) ancestors() []*RClass {
	klasses := []*RClass{c}
	k := c
	for {
		if k.Name == classes.ObjectClass {
			break
		}
		k = k.superClass
		klasses = append(klasses, k)
	}

	return klasses
}

// EqualTo returns true if the class is equal to the given object
func (c *RClass) EqualTo(with Object) bool {
	w, ok := with.(*RClass)
	return ok && c.Name == w.Name && c.class == w.class
}

func (c *RClass) isA(super *RClass) bool {
	cl := c
	for {
		if cl == nil {
			return false
		}
		if cl == super || cl.Name == super.Name {
			return true
		}
		if cl.Name == classes.ObjectClass {
			return false
		}
		cl = cl.superClass
	}
}

func generateSetMethod(attrName string) *BuiltinMethodObject {
	return &BuiltinMethodObject{
		Name: attrName + "=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return receiver.SetVariable("@"+attrName, args[0])
		},
		Primitive: true,
	}
}

func generateGetMethod(attrName string) *BuiltinMethodObject {
	return &BuiltinMethodObject{
		Name: attrName,
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			v, ok := receiver.GetVariable("@" + attrName)
			if ok {
				return v
			}

			return NIL
		},
		Primitive: true,
	}
}

func classIsA(receiver Object, t *Thread, args []Object) Object {
	if len(args) != 1 {
		return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
	}

	rClass, ok := args[0].(*RClass)
	if !ok {
		return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.ClassClass, args[0].Class().Name)
	}

	return BooleanObject(receiver.Class().isA(rClass))
}

func getMethods(klasses []*RClass) (methods []Object) {
	set := map[string]bool{}
	for _, klass := range klasses {
		for _, name := range klass.Methods.names() {
			if !set[name] {
				set[name] = true
				methods = append(methods, StringObject(name))
			}
		}
	}
	return
}
