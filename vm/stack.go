package vm

// Stack is a basic stack implementation
type Stack struct {
	data    []Object
	pointer int
	flags   map[int]bits
}

// Set a value at a given index in the stack.
// TODO: Maybe we should be checking for size before we do this.
// Seems to be safe, but should probably not be exported.
func (s *Stack) Set(index int, o Object) {
	s.data[index] = o
}

func (s *Stack) setTop(o Object) {
	s.data[s.pointer-1] = o
}

// Push an element to the top of the stack
func (s *Stack) Push(v Object) {
	if len(s.data) <= s.pointer {
		s.data = append(s.data, v)
	} else {
		s.data[s.pointer] = v
	}
	s.pointer++
}

// PushFlags an element to the top of the stack
func (s *Stack) PushFlags(v bits) {
	if s.flags == nil {
		s.flags = make(map[int]bits)
	}
	s.flags[s.pointer-1] = v
}

func (s *Stack) topFlags() bits {
	if s.flags == nil {
		return 0
	}
	return s.flags[s.pointer-1]
}

// Pop an element off the top of the stack
func (s *Stack) Pop() Object {
	if len(s.data) < 1 {
		panic("Nothing to pop!")
	}

	if s.pointer < 0 {
		panic("SP is not normal!")
	}

	if s.pointer > 0 {
		s.pointer--
	}

	if s.flags != nil && s.flags[s.pointer] != 0 {
		s.flags[s.pointer] = 0
	}
	v := s.data[s.pointer]
	s.data[s.pointer] = nil
	return v
}

// Discard discard the top of the stack
func (s *Stack) Discard() {
	if s.pointer < 0 {
		panic("SP is not normal!")
	}
	if s.pointer > 0 {
		s.pointer--
	}
	if s.flags != nil {
		s.flags[s.pointer] = 0
	}
	s.data[s.pointer] = nil
}

func (s *Stack) top() Object {
	if len(s.data) != 0 {
		if s.pointer > 0 {
			return s.data[s.pointer-1]
		}
		return s.data[0]
	}
	return nil
}

func (s *Stack) at(pos int) Object {
	if len(s.data) > pos {
		if s.pointer > pos {
			return s.data[s.pointer-1-pos]
		}
		return s.data[0]
	}
	return nil
}
