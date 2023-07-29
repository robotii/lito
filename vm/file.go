package vm

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// FileObject is a special type that contains file pointer so we can keep track on target file.
// Using `File.open` with block is recommended because the instance (block variable) automatically closes.
type FileObject struct {
	BaseObj
	File *os.File
}

var fileModeTable = map[string]int{
	"r":  syscall.O_RDONLY,
	"r+": syscall.O_RDWR,
	"w":  syscall.O_WRONLY,
	"w+": syscall.O_RDWR,
}

var fileClassMethods = []*BuiltinMethodObject{
	{
		Name: "basename",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			fn, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			return StringObject(filepath.Base(string(fn)))
		},
		Primitive: true,
	},
	{
		Name: "chmod",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) < 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentMore, 2, len(args))
			}

			mod, ok := args[0].(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 1, classes.IntegerClass, args[0].Class().Name)
			}

			if !os.FileMode(mod).IsRegular() {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.InvalidChmodNumber, int(mod))
			}

			for i := 1; i < len(args); i++ {
				fn, ok := args[i].(StringObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, i+1, classes.StringClass, args[0].Class().Name)
				}

				filename := string(fn)
				if !filepath.IsAbs(filename) {
					filename = filepath.Join(t.vm.fileDir, filename)
				}

				err := os.Chmod(filename, os.FileMode(uint32(mod)))
				if err != nil {
					return t.vm.InitErrorObject(t, errors.IOError, err.Error())
				}
			}

			return IntegerObject(len(args) - 1)
		},
	},
	{
		Name: "delete",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			for i, arg := range args {
				fn, ok := arg.(StringObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, i+1, classes.StringClass, args[i].Class().Name)
				}
				err := os.Remove(string(fn))

				if err != nil {
					return t.vm.InitErrorObject(t, errors.IOError, err.Error())
				}
			}

			return IntegerObject(len(args))
		},
	},
	{
		Name: "exist?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			fn, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}
			_, err := os.Stat(string(fn))

			return BooleanObject(err == nil)
		},
	},
	{
		Name: "extension",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			fn, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			return StringObject(filepath.Ext(string(fn)))
		},
	},
	{
		Name: "join",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			var e []string
			for i := 0; i < len(args); i++ {
				next, ok := args[i].(StringObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
				}

				e = append(e, string(next))
			}

			return StringObject(filepath.Join(e...))
		},
	},
	{
		Name: "new",
		Fn:   fileNew,
	},
	{
		Name: "open",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			f := fileNew(receiver, t, args)
			file, ok := f.(*FileObject)
			if !ok {
				return f
			}
			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return f
			}
			defer func(File *os.File) {
				_ = File.Close()
			}(file.File)
			return t.Yield(blockFrame, file)
		},
	},
	{
		Name: "size",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			fn, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			filename := string(fn)
			if !filepath.IsAbs(filename) {
				filename = filepath.Join(t.vm.fileDir, filename)
			}

			fs, err := os.Stat(filename)
			if err != nil {
				return t.vm.InitErrorObject(t, errors.IOError, err.Error())
			}

			return IntegerObject(int(fs.Size()))
		},
	},
	{
		Name: "split",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}
			fn, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			d, f := filepath.Split(string(fn))

			return InitArrayObject([]Object{StringObject(d), StringObject(f)})
		},
	},
}

var fileInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "close",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			file := receiver.(*FileObject).File
			_ = file.Close()

			return NIL
		},
	},
	{
		Name: "name",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			name := receiver.(*FileObject).File.Name()
			return StringObject(name)
		},
	},
	{
		Name: "read",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			var result string
			var f []byte
			var err error

			file := receiver.(*FileObject).File

			if file.Name() == "/dev/stdin" {
				reader := bufio.NewReader(os.Stdin)
				result, err = reader.ReadString('\n')
			} else {
				f, err = os.ReadFile(file.Name())
				result = string(f)
			}

			if err != nil && err != io.EOF {
				return t.vm.InitErrorObject(t, errors.IOError, err.Error())
			}

			return StringObject(result)
		},
	},
	{
		Name: "size",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			file := receiver.(*FileObject).File

			fileStats, err := os.Stat(file.Name())
			if err != nil {
				return t.vm.InitErrorObject(t, errors.IOError, err.Error())
			}

			return IntegerObject(int(fileStats.Size()))
		},
	},
	{
		Name: "write",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			// TODO: This needs more validation
			file := receiver.(*FileObject).File
			data := string(args[0].(StringObject))
			length, err := file.Write([]byte(data))

			if err != nil {
				return t.vm.InitErrorObject(t, errors.IOError, err.Error())
			}

			return IntegerObject(length)
		},
	},
}

func initFileObject(vm *VM, f *os.File) *FileObject {
	return &FileObject{
		BaseObj: BaseObj{class: vm.TopLevelClass(classes.FileClass)},
		File:    f,
	}
}

func initFileClass(vm *VM) *RClass {
	return vm.InitClass(classes.FileClass).
		ClassMethods(fileClassMethods).
		InstanceMethods(fileInstanceMethods)
}

// ToString returns the object's name as the string format
func (f *FileObject) ToString(t *Thread) string {
	return "<File: " + f.File.Name() + ">"
}

// Inspect delegates to ToString
func (f *FileObject) Inspect(t *Thread) string {
	return f.ToString(t)
}

// ToJSON just delegates to `ToString`
func (f *FileObject) ToJSON(t *Thread) string {
	return f.ToString(t)
}

// Value returns file object's string format
func (f *FileObject) Value() interface{} {
	return f.File
}

func fileNew(receiver Object, t *Thread, args []Object) Object {
	aLen := len(args)
	if aLen < 1 || aLen > 3 {
		return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentRange, 1, 3, aLen)
	}

	fn, ok := args[0].(StringObject)
	if !ok {
		return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 1, classes.StringClass, args[0].Class().Name)
	}

	mod := syscall.O_RDONLY
	perm := os.FileMode(0755)
	if aLen >= 2 {
		m, ok := args[1].(StringObject)
		if !ok {
			return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 2, classes.StringClass, args[1].Class().Name)
		}

		md, ok := fileModeTable[string(m)]
		if !ok {
			return t.vm.InitErrorObject(t, errors.ArgumentError, "Unknown file mode: %s", string(m))
		}

		if md == syscall.O_RDWR || md == syscall.O_WRONLY {
			_, _ = os.Create(string(fn))
		}

		mod = md
		perm = os.FileMode(0755)

		if aLen == 3 {
			p, ok := args[2].(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 3, classes.IntegerClass, args[2].Class().Name)
			}

			if !os.FileMode(p).IsRegular() {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.InvalidChmodNumber, int(p))
			}

			perm = os.FileMode(p)
		}
	}

	f, err := os.OpenFile(string(fn), mod, perm)

	if err != nil {
		return t.vm.InitErrorObject(t, errors.IOError, err.Error())
	}

	return initFileObject(t.vm, f)
}
