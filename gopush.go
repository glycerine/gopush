package gopush

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// Options holds the configuration options for a Push Interpreter
type Options struct {
	// When TRUE (which is the default), code passed to the top level of
	// the interpreter will be pushed onto the CODE stack prior to
	// execution.
	TopLevelPushCode bool

	// When TRUE, the CODE stack will be popped at the end of top level
	// calls to the interpreter. The default is FALSE.
	TopLevelPopCode bool

	// The maximum number of points that will be executed in a single
	// top-level call to the interpreter.
	EvalPushLimit int

	// The probability that the selection of the ephemeral random NAME
	// constant for inclusion in randomly generated code will produce a new
	// name (rather than a name that was previously generated).
	NewERCNameProbabilty float64

	// The maximum number of points that can occur in any program on the
	// CODE stack. Instructions that would violate this limit act as NOOPs.
	MaxPointsInProgram int

	// The maximum number of points in an expression produced by the
	// CODE.RAND instruction.
	MaxPointsInRandomExpression int

	// The maximum FLOAT that will be produced as an ephemeral random FLOAT
	// constant or from a call to FLOAT.RAND.
	MaxRandomFloat float64

	// The minimum FLOAT that will be produced as an ephemeral random FLOAT
	// constant or from a call to FLOAT.RAND.
	MinRandomFloat float64

	// The maximum INTEGER that will be produced as an ephemeral random
	// INTEGER constant or from a call to INTEGER.RAND.
	MaxRandomInteger int64

	// The minimum INTEGER that will be produced as an ephemeral random
	// INTEGER constant or from a call to INTEGER.RAND.
	MinRandomInteger int64

	// When TRUE the interpreter will print out the stacks after every
	// executed instruction
	Tracing bool

	// A seed for the random number generator.
	RandomSeed int64
}

// Interpreter is a Push interpreter.
type Interpreter struct {
	Stacks      map[string]*Stack
	Options     Options
	Rand        *rand.Rand
	Definitions map[string]Code

	numEvalPush   int
	quoteNextName bool
}

// DefaultOptions hold the default options for the Push Interpreter
var DefaultOptions = Options{
	TopLevelPushCode:            true,
	TopLevelPopCode:             false,
	EvalPushLimit:               1000,
	NewERCNameProbabilty:        0.001,
	MaxPointsInProgram:          100,
	MaxPointsInRandomExpression: 25,
	MaxRandomFloat:              1.0,
	MinRandomFloat:              -1.0,
	MaxRandomInteger:            10,
	MinRandomInteger:            -10,
	Tracing:                     false,
	RandomSeed:                  rand.Int63(),
}

// NewInterpreter returns a new Push Interpreter, configured with the provided Options.
func NewInterpreter(options Options) *Interpreter {
	interpreter := &Interpreter{
		Stacks:        make(map[string]*Stack),
		Options:       options,
		Rand:          rand.New(rand.NewSource(options.RandomSeed)),
		Definitions:   make(map[string]Code),
		numEvalPush:   0,
		quoteNextName: false,
	}

	interpreter.Stacks["integer"] = NewIntStack(interpreter)
	interpreter.Stacks["float"] = NewFloatStack(interpreter)
	interpreter.Stacks["exec"] = NewExecStack(interpreter)
	interpreter.Stacks["code"] = NewCodeStack(interpreter)
	interpreter.Stacks["name"] = NewNameStack(interpreter)
	interpreter.Stacks["boolean"] = NewBooleanStack(interpreter)

	return interpreter
}

func (i *Interpreter) stackOK(name string, mindepth int64) bool {
	s, ok := i.Stacks[name]
	if !ok {
		return false
	}

	if s.Len() < mindepth {
		return false
	}

	return true
}

func (i *Interpreter) printInterpreterState() {
	fmt.Println("Step", i.numEvalPush)
	for k, v := range i.Stacks {
		fmt.Printf("%s:\n", k)
		for i := len(v.Stack) - 1; i >= 0; i-- {
			fmt.Printf("- %v\n", v.Stack[i])
		}
	}
	fmt.Println()
	fmt.Println()
}

func (i *Interpreter) runCode(program Code) (err error) {

	// Recover from a panic that could occur while executing an
	// instruction. Because it is more convenient for functions to not
	// return an error, the functions that want to return an error panic
	// instead.
	defer func() {
		if perr := recover(); perr != nil {
			err = perr.(error)
		}
	}()

	i.Stacks["exec"].Push(program)

	for i.Stacks["exec"].Len() > 0 && i.numEvalPush < i.Options.EvalPushLimit {

		if i.Options.Tracing {
			i.printInterpreterState()
		}

		item := i.Stacks["exec"].Pop().(Code)
		i.numEvalPush++

		// If the item on top of the exec stack is a list, push it in
		// reverse order
		if item.Literal == "" {
			for j := len(item.List) - 1; j >= 0; j-- {
				i.Stacks["exec"].Push(item.List[j])
			}
			continue
		}

		// Try to parse the item on top of the exec stack as a literal
		if intlit, err := strconv.ParseInt(item.Literal, 10, 64); err == nil {
			i.Stacks["integer"].Push(intlit)
			continue
		}

		if floatlit, err := strconv.ParseFloat(item.Literal, 64); err == nil {
			i.Stacks["float"].Push(floatlit)
			continue
		}

		if boollit, err := strconv.ParseBool(item.Literal); err == nil {
			i.Stacks["boolean"].Push(boollit)
			continue
		}

		// Try to parse the item on top of the exec stack as instruction
		if strings.Contains(item.Literal, ".") {
			stack := strings.ToLower(item.Literal[:strings.Index(item.Literal, ".")])
			operation := strings.ToLower(item.Literal[strings.Index(item.Literal, ".")+1:])

			s, ok := i.Stacks[stack]
			if !ok {
				return fmt.Errorf("unknown or disabled stack: %v", stack)
			}

			f, ok := s.Functions[operation]
			if !ok {
				return fmt.Errorf("unknown or disabled instruction %v.%v", stack, operation)
			}

			f()
			continue
		}

		// If the item is not an instruction, it must be a name,
		// either bound or unbound. If the quoteNextName flag is
		// false, we can check if the name is already bound.
		if !i.quoteNextName {
			if d, ok := i.Definitions[strings.ToLower(item.Literal)]; ok {
				// Name is already bound, push its value onto the exec stack
				i.Stacks["exec"].Push(d)
				continue
			}
		}

		// The name is not bound yet, so push it onto the name stack
		i.Stacks["name"].Push(strings.ToLower(item.Literal))
		i.quoteNextName = false
	}

	if i.numEvalPush >= i.Options.EvalPushLimit {
		return errors.New("the EvalPushLimit was exceeded")
	}

	return nil
}

// Run runs the given program written in the Push programming language until
// the EvalPushLimit is reached
func (i *Interpreter) Run(program string) error {
	c, err := ParseCode(program)
	if err != nil {
		return err
	}

	if i.Options.TopLevelPushCode {
		i.Stacks["code"].Push(c)
	}

	err = i.runCode(c)

	if i.Options.TopLevelPopCode {
		i.Stacks["code"].Pop()
	}

	if i.Options.Tracing {
		i.printInterpreterState()
	}

	return err
}
