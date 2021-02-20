package vm

import (
	"github.com/robotii/lito/compiler/bytecode"
	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

func (t *Thread) execFrame(cf *CallFrame) {
	var deferStack []*CallFrame

	// Defer handling - we use a pointer as we will allocate a new slice for
	// deferStack, which will not be captured unless we use the pointer
	defer func(deferStack *[]*CallFrame) {
		// Execute defer statements in reverse order, like Go
		for i := len(*deferStack) - 1; i >= 0; i-- {
			blkFrame := (*deferStack)[i]
			stackPtr := t.Stack.pointer
			// Add something to the stack to prevent overwriting the return value
			t.Stack.Push(NIL)
			// Evaluate the frame
			t.evaluateNormalFrame(blkFrame)
			// Reset the stack pointer
			t.Stack.pointer = stackPtr
		}
	}(&deferStack)

	insCount := cf.instructionsCount()
	stack := &t.Stack
	is := cf.instructionSet
	code := is.Instructions

	for cf.pc < insCount {
		opcode := code[cf.pc]
		t.currentLine = is.SourceMap[cf.pc]
		cf.pc++
	retry:
		switch opcode {

		// Integer bytecodes
		case bytecode.Add, bytecode.Subtract,
			bytecode.Greater, bytecode.GreaterEqual,
			bytecode.Less, bytecode.LessEqual:
			l, lok := stack.at(1).(IntegerObject)
			r, rok := stack.at(0).(IntegerObject)
			if !lok || !rok {
				opcode = bytecode.BinaryOperator
				goto retry
			}
			// Discard the extra parameter
			stack.Discard()
			switch opcode {
			case bytecode.Add:
				stack.setTop(IntegerObject(int(l) + int(r)))

			case bytecode.Subtract:
				stack.setTop(IntegerObject(int(l) - int(r)))

			case bytecode.Less:
				stack.setTop(BooleanObject(int(l) < int(r)))

			case bytecode.Greater:
				stack.setTop(BooleanObject(int(l) > int(r)))

			case bytecode.LessEqual:
				stack.setTop(BooleanObject(int(l) <= int(r)))

			case bytecode.GreaterEqual:
				stack.setTop(BooleanObject(int(l) >= int(r)))
			}
			cf.pc++

		case bytecode.Pop:
			stack.Discard()

		case bytecode.Dup:
			stack.Push(stack.top())

		case bytecode.PutTrue:
			stack.Push(TRUE)

		case bytecode.PutFalse:
			stack.Push(FALSE)

		case bytecode.PutInt:
			stack.Push(IntegerObject(code[cf.pc]))
			cf.pc++

		case bytecode.PutObject:
			stack.Push(t.vm.InitObjectFromGoType(is.GetObject(cf.pc)))
			cf.pc++

		case bytecode.GetConstant, bytecode.GetConstantNamespace:
			constName := is.GetString(cf.pc)
			cf.pc++
			c := t.vm.lookupConstant(t, cf, constName)

			if c == nil {
				t.pushErrorObject(errors.NameError, "uninitialized constant '%s'", constName)
				break
			}

			if stack.top() != nil && (stack.topFlags().has(namespace)) {
				stack.Discard()
			}

			stack.Push(c.Target)
			if opcode == bytecode.GetConstantNamespace {
				stack.PushFlags(namespace)
			}

		case bytecode.GetLocal:
			depth := code[cf.pc]
			cf.pc++
			index := code[cf.pc]
			cf.pc++
			var obj Object

			p := cf.getLocal(index, depth)
			if p == nil || p.Target == nil {
				obj = NIL
			} else {
				obj = p.Target
			}

			stack.Push(obj)

		case bytecode.GetInstanceVariable:
			variableName := is.GetString(cf.pc)
			cf.pc++
			v, ok := cf.self.GetVariable(variableName)
			if !ok {
				stack.Push(NIL)
				break
			}

			stack.Push(v)

		case bytecode.SetInstanceVariable:
			variableName := is.GetString(cf.pc)
			cf.pc++
			p := stack.top()
			cf.self.SetVariable(variableName, p)

		case bytecode.SetOptional:
			p := stack.Pop()
			depth := code[cf.pc]
			cf.pc++
			index := code[cf.pc]
			cf.pc++

			ptr := cf.getLocal(index, depth)
			// We may preallocate these for efficiency
			if ptr == nil || ptr.Target == nil {
				cf.insertLocal(index, depth, p)
			}

		case bytecode.SetLocal:
			p := stack.top()
			depth := code[cf.pc]
			cf.pc++
			index := code[cf.pc]
			cf.pc++

			cf.insertLocal(index, depth, p)

		case bytecode.SetConstant:
			constName := is.GetString(cf.pc)
			cf.pc++
			c := cf.lookupConstantInCurrentScope(constName)
			v := stack.Pop()

			if c != nil {
				t.pushErrorObject(errors.ConstantAlreadyInitializedError, "Constant %s already initialized. Can't assign value to a constant twice.", constName)
			}

			cf.storeConstant(constName, v)

		case bytecode.NewRange, bytecode.NewRangeExcl:
			re := stack.Pop()
			rs := stack.Pop()
			rangeEnd, ok1 := re.(IntegerObject)
			if !ok1 {
				t.pushErrorObject(errors.ArgumentError, errors.WrongArgumentTypeFormat, classes.IntegerClass, re.Class().Name)
			}
			rangeStart, ok2 := rs.(IntegerObject)
			if !ok2 {
				t.pushErrorObject(errors.ArgumentError, errors.WrongArgumentTypeFormat, classes.IntegerClass, rs.Class().Name)
			}
			stack.Push(initRangeObject(t.vm, int(rangeStart), int(rangeEnd), opcode == bytecode.NewRangeExcl))

		case bytecode.NewArray:
			argCount := code[cf.pc]
			cf.pc++

			var elems = make([]Object, argCount)

			for i := argCount - 1; i >= 0; i-- {
				v := stack.Pop()
				elems[i] = v
			}

			arr := InitArrayObject(elems)
			stack.Push(arr)

		case bytecode.ExpandArray:
			arrLength := code[cf.pc]
			cf.pc++
			arr, ok := stack.Pop().(*ArrayObject)

			if !ok {
				t.pushErrorObject(errors.TypeError, "Expect stack top's value to be an Array when executing 'expandarray' instruction.")
			}

			var elems []Object

			for i := 0; i < arrLength; i++ {
				var elem Object
				if i < len(arr.Elements) {
					elem = arr.Elements[i]
				} else {
					elem = NIL
				}

				elems = append([]Object{elem}, elems...)
			}

			for _, elem := range elems {
				stack.Push(elem)
			}

		case bytecode.SplatArray:
			obj := stack.top()
			arr, ok := obj.(*ArrayObject)
			if ok {
				arr.splat = true
			}

		case bytecode.SplatBlock:
			obj := stack.top()
			blk, ok := obj.(*BlockObject)
			if ok {
				blk.splat = true
			}

		case bytecode.NewHash:
			argCount := code[cf.pc]
			cf.pc++
			pairs := map[string]Object{}

			for i := 0; i < argCount/2; i++ {
				v := stack.Pop()
				k := stack.Pop()
				pairs[string(k.(StringObject))] = v
			}
			stack.Push(InitHashObject(pairs))

		case bytecode.BranchUnless:
			v := stack.Pop()
			bo, isBool := v.(BooleanObject)
			if isBool {
				if bo {
					cf.pc++
					break
				}
				cf.pc = code[cf.pc]
				break
			}

			_, isNull := v.(*NilObject)
			if isNull {
				cf.pc = code[cf.pc]
				break
			}
			cf.pc++

		case bytecode.BranchIf:
			v := stack.Pop()
			bo, isBool := v.(BooleanObject)

			if isBool && !bool(bo) {
				cf.pc++
				break
			}

			_, isNull := v.(*NilObject)
			if isNull {
				cf.pc++
				break
			}

			cf.pc = code[cf.pc]
			break

		case bytecode.Jump:
			cf.pc = code[cf.pc]

		case bytecode.Break:
			if cf.IsBlock() {
				frame := t.callFrameStack.pop()
				frame.stopExecution()
				frame.setAsRemoved()
			}

		case bytecode.PutSelf:
			stack.Push(cf.self)

		case bytecode.PutSuper:
			stack.Push(cf.self)
			stack.PushFlags(superRef)

		case bytecode.PutString:
			stack.Push(StringObject(is.GetString(cf.pc)))
			cf.pc++

		case bytecode.PutFloat:
			stack.Push(FloatObject(is.GetFloat(cf.pc)))
			cf.pc++

		case bytecode.PutNull:
			stack.Push(NIL)

		case bytecode.DefMethod:
			argCount := code[cf.pc]
			cf.pc++
			methodName := is.GetString(cf.pc)
			cf.pc++
			is, ok := is.GetObject(cf.pc).(*bytecode.InstructionSet)
			cf.pc++
			if !ok {
				t.pushErrorObject(errors.InternalError, "Can't get method %s's instruction set.", methodName)
			}

			method := &MethodObject{Name: methodName, argc: argCount, instructionSet: is, BaseObj: BaseObj{class: t.vm.TopLevelClass(classes.MethodClass)}}

			v := stack.Pop()
			switch self := v.(type) {
			case *RClass:
				self.Methods[methodName] = method
			default:
				self.Class().Methods[methodName] = method
			}
			// DEBUG: Uncomment this line to write out the method definition
			//os.Stderr.Write([]byte(method.Inspect(t) + "\n"))

		case bytecode.DefMetaMethod:
			argCount := code[cf.pc]
			cf.pc++
			methodName := is.GetString(cf.pc)
			cf.pc++
			is, ok := is.GetObject(cf.pc).(*bytecode.InstructionSet)
			cf.pc++
			if !ok {
				t.pushErrorObject(errors.InternalError, "Can't get method %s's instruction set.", methodName)
			}
			method := &MethodObject{Name: methodName, argc: argCount, instructionSet: is, BaseObj: BaseObj{class: t.vm.TopLevelClass(classes.MethodClass)}}

			v := stack.Pop()
			switch v := v.(type) {
			case *RClass:
				if metaClass := v.MetaClass(); metaClass != nil {
					metaClass.Methods[methodName] = method
				}
			default:
				// TODO: Should we return an error here?
			}
			// DEBUG: Uncomment this line to write out the method definition
			//os.Stderr.Write([]byte(method.Inspect(t) + "\n"))

		case bytecode.DefClass:
			subjectType, subjectName := is.GetString(cf.pc), is.GetString(cf.pc+1)
			cf.pc += 2
			classPtr := cf.lookupConstantUnderAllScope(subjectName)

			classIS := is.GetObject(cf.pc).(*bytecode.InstructionSet)
			cf.pc++

			if classPtr == nil {
				var class *RClass
				if subjectType == "module" {
					class = t.vm.InitModule(subjectName)
				} else {
					class = t.vm.InitClass(subjectName)
				}

				classPtr = cf.storeConstant(class.Name, class)
				superClassName := is.GetString(cf.pc)

				if superClassName != bytecode.NoSuperClass {
					if superClassName == "" {
						t.pushErrorObject(errors.InternalError, "Invalid constant for superclass")
					}
					superClass := t.vm.lookupConstant(t, cf, superClassName)
					inheritedClass, ok := superClass.Target.(*RClass)
					if !ok {
						t.pushErrorObject(errors.InternalError, "Constant %s is not a class. got: %s", superClassName, superClass.Target.Class().Name)
					}

					if inheritedClass.isModule {
						t.pushErrorObject(errors.InternalError, "Module inheritance is not supported: %s", inheritedClass.Name)
					}

					class.inherits(inheritedClass)
				}
			}
			cf.pc++

			stack.Discard()
			c := newNormalCallFrame(classIS, cf.FileName(), t.GetSourceLine())
			c.self = classPtr.Target
			t.evaluateNormalFrame(c)
			stack.Push(classPtr.Target)

		case bytecode.Send:
			methodName := is.GetString(cf.pc)
			cf.pc++
			argCount := code[cf.pc]
			cf.pc++
			blockIS, ok := is.GetObject(cf.pc).(*bytecode.InstructionSet)
			cf.pc++
			if !ok {
				blockIS = nil
			}

			argSet, ok := is.GetObject(cf.pc).(*bytecode.ArgSet)
			cf.pc++

			// Handle splatted block as last argument
			// Check if we have an argument, as we don't want to splat the receiver
			argCount, blockFrame := t.unsplatBlock(cf, argCount, blockIS)

			// Set up the blockframe for execution
			if blockFrame != nil {
				blockFrame.ep = cf
				blockFrame.self = cf.self
				blockFrame.sourceLine = t.GetSourceLine()
			}

			// Deal with splat arguments
			argCount = t.unsplatArray(argCount)

			argPr := stack.pointer - argCount
			receiverPr := argPr - 1
			receiver := stack.data[receiverPr]

			// Find Method
			super := stack.flags[receiverPr].has(superRef)
			t.FindAndExecute(receiver, methodName, super, receiverPr, argPr, argCount, argSet, blockFrame, cf.fileName)

		case bytecode.BinaryOperator:
			methodName := is.GetString(cf.pc)
			cf.pc++
			argCount := 1

			argPr := stack.pointer - argCount
			receiverPr := argPr - 1
			receiver := stack.data[receiverPr]

			// Find Method
			super := stack.flags[receiverPr].has(superRef)
			t.FindAndExecute(receiver, methodName, super, receiverPr, argPr, argCount, nil, nil, cf.fileName)

		case bytecode.InvokeBlock:
			argCount := code[cf.pc]
			cf.pc++
			argPr := stack.pointer - argCount
			receiverPr := argPr - 1
			receiver := stack.data[receiverPr]

			if cf.blockFrame == nil {
				t.pushErrorObject(errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			blockFrame := cf.blockFrame

			if cf.blockFrame.ep == cf.ep {
				blockFrame = cf.blockFrame.ep.blockFrame
			}

			// Check we have a valid block frame still
			if blockFrame == nil {
				t.pushErrorObject(errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			c := newNormalCallFrame(blockFrame.instructionSet, blockFrame.instructionSet.Filename, blockFrame.instructionSet.SourceMap[0])
			c.blockFrame = blockFrame
			c.ep = blockFrame.ep
			c.self = receiver
			c.isBlock = true
			c.initLocalsFrom(stack.data[argPr : argPr+argCount]...)

			t.evaluateNormalFrame(c)

			stack.Set(receiverPr, stack.top())
			stack.pointer = receiverPr + 1

		case bytecode.GetBlock:
			blockFrame := cf.blockFrame
			if blockFrame == nil {
				t.pushErrorObject(errors.InternalError, "Can't get block without a block argument")
			}

			// If the blockframe is from the level up, then reuse that
			if cf.blockFrame.ep == cf.ep && cf.ep != nil && cf.blockFrame.ep.blockFrame != nil {
				blockFrame = cf.blockFrame.ep.blockFrame
			}

			// Check again, just to make sure
			if blockFrame == nil {
				t.pushErrorObject(errors.InternalError, "Can't get block without a block argument")
			}

			blockObject := initBlockObject(t.vm, blockFrame.instructionSet, blockFrame.ep, stack.data[stack.pointer-1])
			stack.Push(blockObject)

		case bytecode.HasBlock:
			stack.Push(BooleanObject(cf.blockFrame != nil))

		case bytecode.Leave:
			cf.stopExecution()

		case bytecode.Defer:
			argCount := code[cf.pc]
			cf.pc++
			blockIS, ok := is.GetObject(cf.pc).(*bytecode.InstructionSet)
			cf.pc++
			if !ok {
				blockIS = nil
			}

			// Allow passing a block as argument
			// Check if we have an argument, as we don't want to splat the receiver
			argCount, blockFrame := t.unsplatBlock(cf, argCount, blockIS)

			// Deal with splat arguments
			argCount = t.unsplatArray(argCount)

			argPr := stack.pointer - argCount
			receiverPr := argPr - 1

			// Set up the blockframe for execution
			if blockFrame != nil {
				blockFrame.ep = cf
				// We take a copy of the receiver as it is at the time
				blockFrame.self = stack.data[receiverPr]
				blockFrame.sourceLine = t.GetSourceLine()

				c := newNormalCallFrame(blockFrame.instructionSet, blockFrame.instructionSet.Filename, t.GetSourceLine())
				c.blockFrame = blockFrame
				c.ep = blockFrame.ep
				c.self = stack.data[receiverPr]
				c.isBlock = true

				// Populate any arguments at the time of the defer call
				// This matches the Go semantics
				c.initLocalsFrom(stack.data[argPr : argPr+argCount]...)

				// Append the call frame to the stack
				deferStack = append(deferStack, c)
			}

			stack.pointer = receiverPr + 1
			// Push something onto the stack for the next instruction
			stack.Push(NIL)

		case bytecode.NoOp:

		default:
			panic("Unexpected bytecode")
		}
	}
}

func (t *Thread) unsplatArray(argCount int) int {
	if arr, ok := t.Stack.top().(*ArrayObject); ok && arr.splat {
		// Pop array
		t.Stack.Discard()
		// Can't count array itself, only the number of array elements
		argCount = argCount - 1 + len(arr.Elements)
		for _, elem := range arr.Elements {
			t.Stack.Push(elem)
		}
	}
	return argCount
}

func (t *Thread) unsplatBlock(cf *CallFrame, argCount int, blockIS *bytecode.InstructionSet) (int, *CallFrame) {
	var blockFrame *CallFrame
	if blk, ok := t.Stack.top().(*BlockObject); ok && argCount > 0 && blk.splat {
		// Pop block
		t.Stack.Discard()
		// Set the blockframe
		blockFrame = blk.asCallFrame(t)
		// This is not considered as an argument
		argCount--
	} else {
		// Find Block
		if blockIS != nil {
			blockFrame = newNormalCallFrame(blockIS, cf.FileName(), cf.SourceLine())
			blockFrame.isBlock = true
		}
	}
	return argCount, blockFrame
}
