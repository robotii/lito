package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/robotii/lito/compiler/bytecode"
	"github.com/robotii/lito/compiler/parser"
	"github.com/robotii/lito/vm/classes"
)

// ClassInitFunc initialises a class for the vm
type ClassInitFunc func(vm *VM) *RClass

// ConfigFunc can be registered with a vm so that it can load this library at vm creation
type ConfigFunc = func(vm *VM) error

// Version stores current Lito version
const Version = "0.2.1"

// FileExt stores the default extension
const FileExt = "lito"

// DefaultLibPath is used for overriding vm.libpath build-time.
var DefaultLibPath string

// LibraryPath set the path to the libraries
func LibraryPath(path string) ConfigFunc {
	return func(vm *VM) error {
		vm.libPath = path
		if vm.libPath == "" {
			vm.libPath = filepath.Join(vm.projectRoot, "lib")
		}
		return nil
	}
}

// Mode sets the mode of the vm
func Mode(mode parser.Mode) ConfigFunc {
	return func(vm *VM) error {
		vm.mode = mode
		return nil
	}
}

var standardClasses = map[string]ClassInitFunc{
	"File":   initFileClass,
	"System": initSystemClass,
}

var baseClasses = map[string]ClassInitFunc{
	"Method":    initMethodClass,
	"Integer":   initIntegerClass,
	"Float":     initFloatClass,
	"String":    initStringClass,
	"Boolean":   initBoolClass,
	"Nil":       initNilClass,
	"Array":     initArrayClass,
	"Hash":      initHashClass,
	"Range":     initRangeClass,
	"Block":     initBlockClass,
	"Channel":   initChannelClass,
	"GoObject":  initGoClass,
	"WaitGroup": initWaitGroupClass,
	"Regexp":    initRegexpClass,
}

var standardLibraries = map[string]func(*VM){
	"json": initJSONClass,
	"lock": initLockClass,
	"spec": initSpecClass,
}

// VM represents a stack based virtual machine.
type VM struct {
	// mainObj is the root object in which all code executes
	mainObj *RObject
	// mainThread contains the main thread of the vm
	mainThread Thread
	// objectClass holds a reference to the Object class, the root of the class hierarchy
	objectClass *RClass
	// errorClass the class of all errors
	errorClass *RClass
	// fileDir indicates executed file's directory
	fileDir string
	// args are command line arguments
	args []string
	// projectRoot holds the root directory of the project
	projectRoot string
	// libPath indicates the libraries path. Defaults to `<projectRoot>/lib`,
	// unless DefaultLibPath is specified.
	libPath string
	// mode holds the parsing mode, as parsing varies when in the REPL
	mode parser.Mode
	// libFiles list of files to be loaded by the vm after initialisation
	libFiles []string
	// threadCount holds the count of threads that have been created.
	// It is never reset.
	threadCount int64
}

// MachineConfigs a list of different machine configurations
var MachineConfigs = map[string]ConfigFunc{
	"standard": standard,
	"sandbox":  sandbox,
}

// New initialises a vm to initial state and returns it.
func New(fileDir string, args []string, configs ...ConfigFunc) (vm *VM, err error) {
	vm = &VM{args: args, threadCount: 1}
	vm.mainThread.vm = vm
	vm.fileDir = fileDir
	vm.projectRoot, _ = filepath.Abs(executableDir())

	_ = LibraryPath(DefaultLibPath)(vm)

	for _, c := range configs {
		if err = c(vm); err != nil {
			return
		}
	}

	if len(configs) == 0 {
		_ = standard(vm)
	}
	vm.mainObj = vm.initMainObj()
	vm.loadLibraryFiles()

	return
}

func (vm *VM) loadLibraryFiles() {
	// Load in the library files requested by the class loaders
	for _, fn := range vm.libFiles {
		err := vm.newThread().loadLibrary(fn)
		if err != nil {
			fmt.Printf("An error occurred when loading lib file %s:\n", fn)
			fmt.Println(err.Error())
		}
	}
}

// AddLibrary adds library files to be loaded by the VM
func (vm *VM) AddLibrary(libs ...string) {
	vm.libFiles = append(vm.libFiles, libs...)
}

func (vm *VM) newThread() (t *Thread) {
	return &Thread{id: atomic.AddInt64(&vm.threadCount, 1), vm: vm}
}

// ExecInstructions accepts a sequence of bytecodes and use vm to evaluate them.
// We also pass in the file name for use in stack traces
func (vm *VM) ExecInstructions(sets []*bytecode.InstructionSet, fn string) {
	program := vm.transferProgram(fn, sets)
	cf := newNormalCallFrame(program, fn, 1)
	cf.self = vm.mainObj

	defer func() {
		switch e := recover().(type) {
		case error:
			panic(e)
		case *Error:
			if vm.mode == parser.NormalMode {
				fmt.Println(e.Message())
			}
			if vm.mode == parser.CommandLineMode {
				fmt.Println(e.Message())
				os.Exit(1)
			}
		}
	}()

	vm.mainThread.evaluateNormalFrame(cf)
}

func (vm *VM) initMainObj() *RObject {
	obj := vm.objectClass.initInstance()

	// TODO: Check if this is still a problem
	// TODO: Fix this as we have broken constants being on the ObjectClass
	/*obj.class = vm.InitClass(fmt.Sprintf("#<Class:%s>", obj.ToString(&vm.mainThread))).
		InstanceMethods(mainObjMetaMethods)
	obj.class.Methods.set("include", vm.TopLevelClass(classes.ClassClass).lookupMethod("include"))
	*/
	return obj
}

func (vm *VM) initConstants() {
	// Init Class and Object
	cClass := initClassClass()
	mClass := initModuleClass(cClass)
	vm.objectClass = initObjectClass(cClass)
	vm.objectClass.SetClassConstant(cClass)
	vm.objectClass.SetClassConstant(mClass)
	vm.objectClass.SetClassConstant(vm.objectClass)
}

// standard initialises a standard VM
func standard(vm *VM) error {
	_ = sandbox(vm)
	for _, initFunc := range standardClasses {
		// Call the init function and store constant
		vm.objectClass.SetClassConstant(initFunc(vm))
	}

	// Init non-sandbox code
	initArgs(vm)
	initEnvironment(vm)
	initStdFiles(vm)

	return nil
}

// sandbox initialises a sandboxed VM
func sandbox(vm *VM) error {
	vm.initConstants()
	// Init builtin classes
	for _, initFunc := range baseClasses {
		// Call the init function and store constant
		vm.objectClass.SetClassConstant(initFunc(vm))
	}

	// Init error classes
	initErrorClasses(vm)

	return nil
}

// Init Env
func initEnvironment(vm *VM) {
	envs := map[string]Object{}

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envs[pair[0]] = StringObject(pair[1])
	}
	vm.objectClass.constants["Env"] = &Pointer{Target: InitHashObject(envs)}
}

func initArgs(vm *VM) {
	// Init Args
	args := make([]Object, 0, len(vm.args))
	for _, arg := range vm.args {
		args = append(args, StringObject(arg))
	}
	vm.objectClass.constants["Args"] = &Pointer{Target: InitArrayObject(args)}
}

func initStdFiles(vm *VM) {
	vm.objectClass.constants["Stdout"] = &Pointer{Target: initFileObject(vm, os.Stdout)}
	vm.objectClass.constants["Stderr"] = &Pointer{Target: initFileObject(vm, os.Stderr)}
	vm.objectClass.constants["Stdin"] = &Pointer{Target: initFileObject(vm, os.Stdin)}
}

// TopLevelClass returns the class for a given name
func (vm *VM) TopLevelClass(cn string) *RClass {
	objClass := vm.objectClass

	if cn == classes.ObjectClass {
		return objClass
	}

	return objClass.constants[cn].Target.(*RClass)
}

// CurrentFilePath returns the current file name
func (vm *VM) CurrentFilePath() string {
	frame := vm.mainThread.callFrameStack.top()
	return frame.FileName()
}

// loadConstant makes sure we don't create a class twice.
func (vm *VM) loadConstant(name string, isModule bool) *RClass {
	var c *RClass

	ptr := vm.objectClass.constants[name]
	if ptr != nil {
		return ptr.Target.(*RClass)
	}
	if isModule {
		c = vm.InitModule(name)
	} else {
		c = vm.InitClass(name)
	}

	vm.objectClass.SetClassConstant(c)
	return c
}

func (vm *VM) lookupConstant(t *Thread, cf callFrame, constName string) (constant *Pointer) {
	var namespace *RClass
	var hasNamespace bool

	top := t.Stack.top()

	if top != nil {
		namespace, hasNamespace = top.(*RClass)
	}

	if hasNamespace && namespace != nil {
		constant = namespace.lookupConstantUnderAllScope(constName)

		if constant != nil {
			return
		}
	}

	constant = cf.lookupConstantUnderAllScope(constName)
	if constant == nil {
		constant = vm.objectClass.constants[constName]
	}

	if constName == classes.ObjectClass {
		constant = &Pointer{Target: vm.objectClass}
	}

	return
}

// executableDir returns the directory of the current executable
func executableDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}

// transferProgram transfers the instruction sets into the VM and returns the main program
func (vm *VM) transferProgram(filename string, sets []*bytecode.InstructionSet) *bytecode.InstructionSet {
	var program *bytecode.InstructionSet
	for _, set := range sets {
		// Set the filename for each instruction set
		set.Filename = filename
		switch set.Type {
		case bytecode.Program:
			program = set
		}
	}

	return program
}
