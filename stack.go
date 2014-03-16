package gopush

type Instruction func(map[string]*Stack)

type Stack struct {
	Stack     []interface{}
	Functions map[string]Instruction
}

func (s Stack) Peek() interface{} {
	if len(s.Stack) == 0 {
		return struct{}{}
	}

	return s.Stack[len(s.Stack)-1]
}

func (s *Stack) Push(lit interface{}) {
	s.Stack = append(s.Stack, lit)
}

func (s *Stack) Pop() (item interface{}) {
	if len(s.Stack) == 0 {
		return struct{}{}
	}

	item = s.Stack[len(s.Stack)-1]
	s.Stack = s.Stack[:len(s.Stack)-1]

	return item
}

func (s Stack) Len() int64 {
	return int64(len(s.Stack))
}

func (s *Stack) Dup() {
	if len(s.Stack) == 0 {
		return
	}

	s.Push(s.Peek())
}

func (s *Stack) Swap() {
	if len(s.Stack) < 2 {
		return
	}

	i1 := s.Pop()
	i2 := s.Pop()
	s.Push(i1)
	s.Push(i2)
}

func (s *Stack) Flush() {
	s.Stack = nil
}

func (s *Stack) Rot() {
	if len(s.Stack) < 3 {
		return
	}

	i1 := s.Pop()
	i2 := s.Pop()
	i3 := s.Pop()

	s.Push(i2)
	s.Push(i1)
	s.Push(i3)
}

func (s *Stack) Shove(item interface{}, idx int64) {
	//TODO
}

func (s *Stack) Yank(idx int64) {
	// TODO
}

func (s *Stack) YankDup(idx int64) {
	// TODO
}