package repl

import (
	"fmt"
	"regexp"

	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/robotii/lito/compiler/bytecode"
	"github.com/robotii/lito/compiler/lexer"
	"github.com/robotii/lito/compiler/parser"
	"github.com/robotii/lito/compiler/parser/errors"
	"github.com/robotii/lito/fsm"
	"github.com/robotii/lito/vm"

	"github.com/chzyer/readline"
)

const (
	prmpt1  = "."
	prmpt2  = ":"
	prompt1 = "\033[32m" + prmpt1 + "\033[0m "
	prompt2 = "\033[34m" + prmpt2 + "\033[0m "
	pad     = "  "
	echo    = "\033[33m#>\033[0m"
	redB    = "\033[31m"
	redE    = "\033[0m"

	help  = ".help"
	reset = ".reset"

	semicolon = ';'

	readyToExec = "readyToExec"
	waiting     = "waiting"
	waitEnded   = "waitEnded"
)

// iREPL holds internal states of iREPL.
type iREPL struct {
	sm         *fsm.FSM
	rl         *readline.Instance
	line       string
	cmds       []string
	indent     int
	prevIndent int
	inspect    bool
}

// iVM holds VM only for iREPL.
type iVM struct {
	vm     *vm.VM
	parser *parser.Parser
	gen    *bytecode.Generator
}

var out io.Writer

func init() {
	out = os.Stderr
}

func oprintln(s ...string) {
	_, _ = out.Write([]byte(strings.Join(s, " ") + "\n"))
}

// StartREPL starts Lito's REPL.
func StartREPL(version string, inspect bool, mType string) {

reset:
	var err error
	repl := newREPL(inspect)

	repl.rl, err = readline.NewEx(&readline.Config{
		Prompt:              prompt1,
		HistoryFile:         filepath.Join(os.TempDir(), "readline_lito.tmp"),
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	defer func() { _ = repl.rl.Close() }()
	if err != nil {
		fmt.Printf("REPL init error: %s", err)
		return
	}

	oprintln("lito", version)

	ivm, err := newIVM(mType)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for {
		err = repl.read()
		// Interruption handling
		if err != nil {
			switch err {
			case io.EOF:
				oprintln(repl.line + "")
				return
			case readline.ErrInterrupt: // Pressing Ctrl-C
				if len(repl.line) == 0 {
					if repl.cmds == nil {
						return
					}
				}
				// Erasing command buffer
				repl.eraseBuffer()
				continue
			}
		}

		// Handle comments
		if strings.HasPrefix(repl.line, "#") {
			oprintln(switchPrompt(repl.indent) + indent(repl.prevIndent) + repl.line)
			continue
		}

		// Command handling
		switch repl.line {
		case help:
			oprintln(switchPrompt(repl.indent) + repl.line)
			usage(repl.rl.Stderr())
			continue
		case reset:
			repl.rl = nil
			repl.cmds = nil
			oprintln(switchPrompt(repl.indent) + repl.line)
			oprintln("Restarting REPL...")
			goto reset
		case "":
			oprintln(switchPrompt(repl.indent) + indent(repl.indent) + repl.line)
			continue
		}

		// Concatenate previous input here
		repl.cmds = append(repl.cmds, repl.line)
		input := strings.Join(repl.cmds, "\n")
		repl.prevIndent = repl.indent
		repl.indent = strings.Count(input, "{") - strings.Count(input, "}")

		// Multi-line quotation handling
		dq := checkOpenQuotes(input, `"`, `'`)
		sq := checkOpenQuotes(input, `'`, `"`)
		// Handle open quotes
		if dq || sq {
			oprintln(switchPrompt(repl.indent) + indent(repl.prevIndent) + repl.line)
			repl.rl.SetPrompt(prompt2)
			continue
		}

		// Parsing
		ivm.parser.Lexer = lexer.New(input)
		program, pErr := ivm.parser.ParseProgram()

		// Parse error handling
		if pErr != nil {
			switch {
			// To handle 'switch'
			case pErr.IsUnexpectedSwitch():
				oprintln(switchPrompt(repl.indent) + indent(repl.prevIndent) + repl.line)
				repl.rl.SetPrompt(prompt2 + indent(repl.indent))
				repl.sm.State(waiting)
				continue

			// To handle such as 'else' or 'elsif'
			// The prompt should be `:` even on the top level indentation when the line is `else` or `elif` or like that
			case pErr.IsUnexpectedToken():
				oprintln(prompt2 + indent(repl.indent-1) + repl.line)
				repl.rl.SetPrompt(prompt2 + indent(repl.indent))
				repl.sm.State(waiting)
				continue

			// To handle beginning of a block
			case pErr.IsUnexpectedEOF():
				if !repl.sm.Is(waiting) {
					repl.sm.State(waiting)
				}
				repl.printAndIndent()
				continue

			// To handle '}'
			case pErr.IsUnexpectedEnd():
				// Exiting error handling
				repl.indent = 0
				repl.prevIndent = 0
				repl.rl.SetPrompt(switchPrompt(repl.indent) + indent(repl.indent))
				repl.sm.State(waitEnded)
			}
		}

		if repl.sm.Is(waiting) {
			if repl.indent <= 0 {
				repl.sm.State(waitEnded)
			} else { // Still indented
				oprintln(switchPrompt(repl.indent) + indent(repl.prevIndent) + repl.line)
				repl.rl.SetPrompt(switchPrompt(repl.indent) + indent(repl.indent))
				continue
			}
		}

		// Ending the block and prepare execution
		if repl.sm.Is(waitEnded) {
			ivm.parser.Lexer = lexer.New(strings.Join(repl.cmds, "\n"))

			// Test if current input can be properly parsed.
			program, pErr = ivm.parser.ParseProgram()

			if pErr != nil {
				repl.handleParserError(pErr)
				continue
			}

			// If everything goes well, reset state and statements buffer
			repl.rl.SetPrompt(switchPrompt(repl.indent))
			repl.sm.State(readyToExec)
		}

		// Execute the lines
		if repl.sm.Is(readyToExec) {
			oprintln(switchPrompt(repl.prevIndent) + repl.line)

			if pErr != nil {
				repl.handleParserError(pErr)
				continue
			}

			// Keep the blocks but discard methods,
			// as these are already transferred onto the classes
			ivm.gen.ResetMethodInstructionSets()
			currentISI := ivm.gen.Index()
			instructions := ivm.gen.GenerateInstructions(program.Statements)
			ivm.vm.REPLExec(instructions)

			var r string
			o := ivm.vm.GetExecResult()
			if errObj, ok := o.(*vm.Error); ok && errObj.Raised {
				r = redB + errObj.ToString(nil) + redE
				ivm.vm.GetREPLResult() // Discard
			} else {
				r = ivm.vm.GetREPLResult()
			}

			// Suppress echo back on trailing ';'
			if t := repl.cmds[len(repl.cmds)-1]; t[len(t)-1] != semicolon {
				oprintln(echo, r)
			}

			if repl.inspect {
				for _, is := range instructions[currentISI:] {
					oprintln(is.Inspect())
				}
			}

			// Clear the commands
			repl.cmds = nil
			// Set the prompt back to normal
			repl.rl.SetPrompt(prompt1)
		}
	}
}

func (repl *iREPL) handleParserError(e *errors.Error) {
	if e != nil {
		if !e.IsEOF() {
			fmt.Println(e.Message)
		}
		oprintln(switchPrompt(repl.indent) + indent(repl.prevIndent) + repl.line)
	}
	repl.eraseBuffer()
}

func (repl *iREPL) eraseBuffer() {
	repl.indent = 0
	repl.prevIndent = 0
	repl.rl.SetPrompt(prompt1)
	repl.sm.State(waiting)
	repl.sm.State(readyToExec)
	repl.cmds = nil
}

// Prints and add an indent.
func (repl *iREPL) printAndIndent() {
	if strings.HasPrefix(repl.line, "}") {
		repl.prevIndent--
	}
	oprintln(switchPrompt(repl.indent+1) + indent(repl.prevIndent) + repl.line)
	repl.rl.SetPrompt(switchPrompt(repl.indent+1) + indent(repl.indent))
}

// filterInput just ignores Ctrl-z.
func filterInput(r rune) (rune, bool) {
	return r, r != readline.CharCtrlZ
}

// indent performs indentation with space padding.
func indent(c int) string {
	if c <= 0 {
		return ""
	}
	return strings.Repeat(pad, c)
}

// newREPL creates a new iREPL.
func newREPL(inspect bool) *iREPL {
	return &iREPL{
		sm: fsm.New(
			readyToExec,
			fsm.States{
				{Name: waiting, From: []string{waitEnded, readyToExec}},
				{Name: waitEnded, From: []string{waiting}},
				{Name: readyToExec, From: []string{waitEnded, readyToExec, waiting}},
			},
		),
		inspect: inspect,
	}
}

// newIVM creates a new iVM.
func newIVM(mType string) (ivm iVM, err error) {
	var configs []vm.ConfigFunc
	if cfg, ok := vm.MachineConfigs[mType]; ok {
		configs = append(configs, cfg)
	}
	ivm = iVM{}
	ivm.vm, err = vm.New("", []string{}, configs...)
	if err == nil {
		ivm.vm.InitForREPL()

		// Initialize parser, lexer is not important here
		ivm.parser = parser.New(lexer.New(""), parser.REPLMode)
		program, _ := ivm.parser.ParseProgram()
		// Initialize code generator, and it will behave a little different in REPL mode.
		ivm.gen = bytecode.NewGenerator()
		ivm.gen.REPL = true
		ivm.gen.InitTopLevelScope(program)
	}
	return ivm, err
}

// Returns true if the specified quotation in the string is open.
func checkOpenQuotes(s, open, ignore string) bool {
	s = strings.ReplaceAll(s, "\\\\", "")
	s = strings.ReplaceAll(s, "\\\"", "")
	s = strings.ReplaceAll(s, "\\'", "")

	rq := regexp.MustCompile(`^[^` + ignore + `]*` + open)
	isOpen := rq.MatchString(s)
	if strings.Count(s, open)%2 == 1 && isOpen {
		return true
	}
	return false
}

// switchPrompt switches the prompt sign.
func switchPrompt(s int) string {
	if s > 0 {
		return prompt2
	}
	return prompt1
}

// read fetches one line from input, with the help of Readline library.
func (repl *iREPL) read() error {
	repl.rl.Config.UniqueEditLine = true // required to update the prompt
	line, err := repl.rl.Readline()
	repl.rl.Config.UniqueEditLine = false
	line = strings.TrimSpace(line)
	repl.line = line
	return err
}

// usage shows help lines.
func usage(w io.Writer) {
	_, _ = io.WriteString(w, "commands:\n   .help\n   .reset\n")
}
