package vm

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/robotii/lito/vm/classes"
	"github.com/robotii/lito/vm/errors"
)

// StringObject represents string instances.
type StringObject string

var stringClass *RClass

var stringClassMethods = []*BuiltinMethodObject{
	{
		Name: "fmt",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) < 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentMore, 1, len(args))
			}

			formatObj, ok := args[0].(StringObject)

			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			format := string(formatObj)
			var arguments []interface{}

			for _, arg := range args[1:] {
				arguments = append(arguments, arg.ToString(t))
			}

			count := strings.Count(format, "%s")

			if len(args[1:]) != count {
				return t.vm.InitErrorObject(t, errors.ArgumentError, "Expect %d additional string(s) to insert. got: %d", count, len(args[1:]))
			}

			return StringObject(fmt.Sprintf(format, arguments...))
		},
	},
	{
		Name: "new",
		Fn:   NoSuchMethod("new"),
	},
}

var stringInstanceMethods = []*BuiltinMethodObject{
	{
		Name: "+",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			right, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			left := receiver.(StringObject)
			return StringObject(string(left) + string(right))
		},
	},
	{
		Name: "*",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			right, ok := args[0].(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
			}

			if int(right) < 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.NegativeSecondValue, int(right))
			}

			left := receiver.(StringObject)
			return StringObject(strings.Repeat(string(left), int(right)))
		},
	},
	{
		Name: ">",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			right, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			left := receiver.(StringObject)
			return BooleanObject(string(left) > string(right))
		},
	},
	{
		Name: "<",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			right, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			left := receiver.(StringObject)
			return BooleanObject(string(left) < string(right))
		},
	},
	{
		Name: "!=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			right, ok := args[0].(StringObject)
			if !ok {
				return TRUE
			}

			left := receiver.(StringObject)
			return BooleanObject(string(left) != string(right))
		},
	},
	{
		Name: "[]",
		Fn:   stringSlice,
	},
	{
		Name: "[]=",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 2, len(args))
			}

			index, ok := args[0].(IntegerObject)

			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.IntegerClass, args[0].Class().Name)
			}

			indexValue := int(index)
			str := string(receiver.(StringObject))
			strLength := utf8.RuneCountInString(str)

			if strLength < indexValue {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.IndexOutOfRange, strconv.Itoa(indexValue))
			}

			replaceStr, ok := args[1].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[1].Class().Name)
			}
			replaceStrValue := string(replaceStr)

			// Negative Index Case
			if indexValue < 0 {
				if -indexValue > strLength {
					return t.vm.InitErrorObject(t, errors.ArgumentError, errors.IndexOutOfRange, strconv.Itoa(indexValue))
				}
				// Change to positive index to replace the string
				indexValue += strLength
			}

			if strLength == indexValue {
				return StringObject(str + replaceStrValue)
			}
			result := string([]rune(str)[:indexValue]) + replaceStrValue + string([]rune(str)[indexValue+1:])
			return StringObject(result)
		},
	},
	{
		Name: "capitalise",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			str := string(receiver.(StringObject))
			start := string([]rune(str)[0])
			rest := string([]rune(str)[1:])
			result := strings.ToUpper(start) + strings.ToLower(rest)
			return StringObject(result)
		},
	},
	{
		Name: "chop",
		Fn: func(receiver Object, t *Thread, args []Object) Object {

			str := string(receiver.(StringObject))
			strLength := utf8.RuneCountInString(str)

			return StringObject([]rune(str)[:strLength-1])
		},
	},
	{
		Name: "count",
		Fn:   stringSize,
	},
	{
		Name: "delete",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			deleteStr, ok := args[0].(StringObject)

			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			str := string(receiver.(StringObject))
			return StringObject(strings.Replace(str, string(deleteStr), "", -1))
		},
	},
	{
		Name: "lower",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return StringObject(strings.ToLower(string(receiver.(StringObject))))
		},
	},
	{
		Name: "dup",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return receiver
		},
	},
	{
		Name: "each_byte",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			str := string(receiver.(StringObject))
			if blockFrame.IsEmpty() {
				return StringObject(str)
			}

			for _, b := range []byte(str) {
				t.Yield(blockFrame, IntegerObject(int(b)))
			}

			return StringObject(str)
		},
	},
	{
		Name: "each_char",
		Fn:   strEachChar,
	},
	{
		Name: "each",
		Fn:   strEachChar,
	},
	{
		Name: "each_line",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			blockFrame := t.GetBlock()
			if blockFrame == nil {
				return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
			}

			str := string(receiver.(StringObject))
			if blockFrame.IsEmpty() {
				return StringObject(str)
			}
			lineArray := strings.Split(str, "\n")

			for _, line := range lineArray {
				t.Yield(blockFrame, StringObject(line))
			}

			return StringObject(str)
		},
	},
	{
		Name: "empty?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			str := string(receiver.(StringObject))
			return BooleanObject(str == "")
		},
		Primitive: true,
	},
	{
		Name: "ends_with?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			compareStr, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			compareStrValue := string(compareStr)
			str := string(receiver.(StringObject))

			return BooleanObject(strings.HasSuffix(str, compareStrValue))
		},
	},
	{
		Name: "include?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			includeStr, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			str := string(receiver.(StringObject))
			return BooleanObject(strings.Contains(str, string(includeStr)))
		},
	},
	{
		Name: "insert",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 2, len(args))
			}

			index, ok := args[0].(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 1, classes.IntegerClass, args[0].Class().Name)
			}

			insertStr, ok := args[1].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 2, classes.StringClass, args[1].Class().Name)
			}

			indexValue := int(index)
			str := string(receiver.(StringObject))
			strLength := utf8.RuneCountInString(str)

			if indexValue < 0 {
				if -indexValue > strLength+1 {
					return t.vm.InitErrorObject(t, errors.ArgumentError, errors.IndexOutOfRange, indexValue)
				} else if -indexValue == strLength+1 {
					return StringObject(string(insertStr) + str)
				}
				// Change it to positive index value to replace the string via index
				indexValue += strLength
			}

			if strLength < indexValue || indexValue < 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.IndexOutOfRange, indexValue)
			}

			return StringObject(string([]rune(str)[:indexValue]) + string(insertStr) + string([]rune(str)[indexValue:]))
		},
	},
	{
		Name: "length",
		Fn:   stringSize,
	},
	{
		Name: "ljust",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			aLen := len(args)
			if aLen < 1 || aLen > 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentRange, 1, 2, aLen)
			}

			strLength, ok := args[0].(IntegerObject)

			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 1, classes.IntegerClass, args[0].Class().Name)
			}

			strLengthValue := int(strLength)

			var padStrValue string
			if aLen == 1 {
				padStrValue = " "
			} else {
				p := args[1]
				padStr, ok := p.(StringObject)

				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 2, classes.StringClass, p.Class().Name)
				}

				padStrValue = string(padStr)
			}

			str := string(receiver.(StringObject))
			currentStrLength := utf8.RuneCountInString(str)
			padStrLength := utf8.RuneCountInString(padStrValue)

			if strLengthValue > currentStrLength {
				for i := currentStrLength; i < strLengthValue; i += padStrLength {
					str += padStrValue
				}
				str = string([]rune(str)[:strLengthValue])
			}

			return StringObject(str)
		},
	},
	{
		Name: "replace",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 2, len(args))
			}

			r := args[1]
			replacement, ok := r.(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 2, classes.StringClass, args[1].Class().Name)
			}

			target := string(receiver.(StringObject))
			switch pattern := args[0].(type) {
			case StringObject:
				return StringObject(strings.Replace(target, string(pattern), string(replacement), -1))
			default:
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 1, classes.StringClass+" or "+classes.RegexpClass, args[0].Class().Name)
			}
		},
	},
	{
		Name: "replace_once",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 2, len(args))
			}

			r := args[1]
			replacement, ok := r.(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 2, classes.StringClass, args[1].Class().Name)
			}

			target := string(receiver.(StringObject))
			switch pattern := args[0].(type) {
			case StringObject:
				return StringObject(strings.Replace(target, string(pattern), string(replacement), 1))
			default:
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 1, classes.StringClass+" or "+classes.RegexpClass, args[0].Class().Name)
			}
		},
	},
	{
		Name: "reverse",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			str := string(receiver.(StringObject))
			return StringObject(reverseString(str))
		},
	},
	{
		Name: "rjust",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			aLen := len(args)
			if aLen < 1 || aLen > 2 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgumentRange, 1, 2, aLen)
			}

			strLength, ok := args[0].(IntegerObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 1, classes.IntegerClass, args[0].Class().Name)
			}

			strLengthValue := int(strLength)

			var padStrValue string
			if aLen == 1 {
				padStrValue = " "
			} else {
				p := args[1]
				padStr, ok := p.(StringObject)
				if !ok {
					return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormatNum, 2, classes.StringClass, args[1].Class().Name)
				}

				padStrValue = string(padStr)
			}

			padStrLength := utf8.RuneCountInString(padStrValue)

			str := string(receiver.(StringObject))
			if strLengthValue > len(str) {
				origin := str
				originStrLength := utf8.RuneCountInString(origin)
				for i := originStrLength; i < strLengthValue; i += padStrLength {
					str = padStrValue + str
				}
				currentStrLength := utf8.RuneCountInString(str)
				if currentStrLength > strLengthValue {
					chopLength := currentStrLength - strLengthValue
					str = string([]rune(str)[:currentStrLength-originStrLength-chopLength]) + origin
				}
			}

			return StringObject(str)
		},
	},
	{
		Name: "size",
		Fn:   stringSize,
	},
	{
		Name: "slice",
		Fn:   stringSlice,
	},
	{
		Name: "split",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			separator, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			str := string(receiver.(StringObject))
			return StringObjectSplit(t.vm, str, string(separator))
		},
	},
	{
		Name: "starts_with?",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 1 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
			}

			compareStr, ok := args[0].(StringObject)
			if !ok {
				return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, classes.StringClass, args[0].Class().Name)
			}

			compareStrValue := string(compareStr)
			str := string(receiver.(StringObject))

			return BooleanObject(strings.HasPrefix(str, compareStrValue))
		},
	},
	{
		Name: "strip",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			str := string(receiver.(StringObject))
			str = strings.Trim(str, " \n\t\r\v")
			return StringObject(str)
		},
	},
	{
		Name: "array",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			str := receiver.(StringObject)
			return StringObjectSplit(t.vm, string(str), "")
		},
	},
	{
		Name: "bytes",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return initGoObject(t.vm, []byte(receiver.(StringObject)))
		},
	},
	{
		Name: "float",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			str := string(receiver.(StringObject))

			for i, char := range str {
				if !unicode.IsSpace(char) {
					str = str[i:]
					break
				}
			}

			parsedStr, ok := strconv.ParseFloat(str, 64)

			if ok != nil {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.InvalidNumericString, str)
			}

			return FloatObject(parsedStr)
		},
	},
	{
		Name: "int",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			str := string(receiver.(StringObject))
			parsedStr, err := strconv.ParseInt(str, 10, 0)

			if err == nil {
				return IntegerObject(int(parsedStr))
			}

			var digits string
			for _, char := range str {
				if unicode.IsDigit(char) {
					digits += string(char)
				} else if unicode.IsSpace(char) && len(digits) == 0 {
					// do nothing; allow trailing spaces
				} else {
					break
				}
			}

			if len(digits) > 0 {
				parsedStr, _ = strconv.ParseInt(digits, 10, 0)
				return IntegerObject(int(parsedStr))
			}

			return IntegerObject(0)
		},
	},
	{
		Name: "string",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}
			return receiver
		},
	},
	{
		Name: "inspect",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return StringObject(receiver.(StringObject).Inspect(t))
		},
	},
	{
		Name: "upper",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			return StringObject(strings.ToUpper(string(receiver.(StringObject))))
		},
	},
	{
		Name: "json",
		Fn: func(receiver Object, t *Thread, args []Object) Object {
			if len(args) != 0 {
				return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
			}

			r := receiver.(StringObject)
			return StringObject(r.ToJSON(t))
		},
	},
}

func initStringClass(vm *VM) *RClass {
	stringClass = vm.InitClass(classes.StringClass).
		ClassMethods(stringClassMethods).
		InstanceMethods(stringInstanceMethods)
	return stringClass
}

// Value returns the object
func (s StringObject) Value() interface{} {
	return string(s)
}

// ToString returns the object's name as the string format
func (s StringObject) ToString(t *Thread) string {
	return fmt.Sprintf(`%s`, string(s))
}

// Inspect wraps ToString with double quotes
func (s StringObject) Inspect(t *Thread) string {
	return fmt.Sprintf(`"%s"`, escapeSpecialChars(escapeBackslash(s.ToString(t))))
}

// EqualTo returns if the StringObject is equal to another object
func (s StringObject) EqualTo(compared Object) bool {
	right, ok := compared.(StringObject)
	return ok && s.equal(right)
}

func escapeSpecialChars(s string) string {
	s = strings.Replace(s, "\n", `\n`, -1)
	s = strings.Replace(s, `"`, `\"`, -1)
	return s
}

func escapeBackslash(s string) string {
	return strings.Replace(s, `\`, `\\`, -1)
}

// ToJSON just delegates to ToString
func (s StringObject) ToJSON(t *Thread) string {
	return strconv.Quote(string(s))
}

// equal returns true if the String values between receiver and parameter are equal
func (s StringObject) equal(e StringObject) bool {
	return string(s) == string(e)
}

// StringObjectSplit returns an ArrayObject with the split strings
func StringObjectSplit(vm *VM, s string, sep string) *ArrayObject {
	arr := strings.Split(s, sep)
	elements := make([]Object, len(arr))
	for i := 0; i < len(arr); i++ {
		elements[i] = StringObject(arr[i])
	}
	return InitArrayObject(elements)
}

// Class ...
func (s StringObject) Class() *RClass {
	return stringClass
}

// GetVariable ...
func (s StringObject) GetVariable(string) (Object, bool) {
	return nil, false
}

// SetVariable ...
func (s StringObject) SetVariable(n string, o Object) Object {
	return o
}

// FindLookup ...
func (s StringObject) FindLookup(searchAncestor bool) (method Object) {
	method, _ = s.Class().Methods[lookupMethod]
	if method == nil && searchAncestor {
		method = s.FindMethod(lookupMethod, false)
	}

	return
}

// FindMethod ...
func (s StringObject) FindMethod(methodName string, super bool) (method Object) {
	if super {
		return s.Class().superClass.lookupMethod(methodName)
	}
	return s.Class().lookupMethod(methodName)
}

// Variables ...
func (s StringObject) Variables() Environment {
	return nil
}

// SetVariables ...
func (s StringObject) SetVariables(Environment) {
}

// IsTruthy ...
func (s StringObject) IsTruthy() bool {
	return string(s) != ""
}

func stringSlice(receiver Object, t *Thread, args []Object) Object {
	if len(args) != 1 {
		return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 1, len(args))
	}

	str := string(receiver.(StringObject))

	switch slice := args[0].(type) {
	case *RangeObject:
		return sliceByRange(str, slice)

	case IntegerObject:
		return sliceByInt(str, int(slice))

	default:
		return t.vm.InitErrorObject(t, errors.TypeError, errors.WrongArgumentTypeFormat, "Range or Integer", slice.Class().Name)
	}
}

func sliceByRange(str string, ro *RangeObject) Object {
	inc := 1
	if ro.Exclusive {
		inc = 0
	}

	start, startOk := normaliseStringIndex(str, ro.Start, 0)
	end, endOk := normaliseStringIndex(str, ro.End, inc)
	if startOk && endOk && start <= end {
		//return StringObject(fmt.Sprint(start, endOk))
		return StringObject([]rune(str)[start:end])
	}

	return NIL
}

func sliceByInt(str string, iv int) Object {
	iv, ok := normaliseStringIndex(str, iv, 0)
	if !ok {
		return nil
	}
	return StringObject([]rune(str)[iv])
}

func normaliseStringIndex(str string, iv int, inc int) (int, bool) {
	strLength := utf8.RuneCountInString(str)
	if iv < 0 {
		iv += strLength
		if iv+inc < 0 {
			return 0, false
		}
	}
	if iv+inc > strLength {
		return 0, false
	}
	return iv + inc, true
}

func stringSize(receiver Object, t *Thread, args []Object) Object {
	str := string(receiver.(StringObject))

	return IntegerObject(utf8.RuneCountInString(str))
}

func strEachChar(receiver Object, t *Thread, args []Object) Object {
	if len(args) != 0 {
		return t.vm.InitErrorObject(t, errors.ArgumentError, errors.WrongNumberOfArgument, 0, len(args))
	}

	blockFrame := t.GetBlock()
	if blockFrame == nil {
		return t.vm.InitErrorObject(t, errors.InternalError, errors.CantYieldWithoutBlockFormat)
	}

	str := string(receiver.(StringObject))
	if blockFrame.IsEmpty() {
		return StringObject(str)
	}

	for _, char := range []rune(str) {
		t.Yield(blockFrame, StringObject(char))
	}

	return StringObject(str)
}

// reverseString interprets its argument as UTF-8
// and ignores bytes that do not form valid UTF-8.  return value is UTF-8.
func reverseString(str string) string {
	if str == "" {
		return ""
	}
	srcRunes := []rune(str)
	targetRunes := make([]rune, len(srcRunes))
	start := len(targetRunes)
	for i := 0; i < len(srcRunes); {
		// Skip invalid characters
		if srcRunes[i] == utf8.RuneError {
			i++
			continue
		}
		// Check if the next character should be combined
		j := i + 1
		for j < len(srcRunes) && (unicode.Is(unicode.Mn, srcRunes[j]) ||
			unicode.Is(unicode.Me, srcRunes[j]) || unicode.Is(unicode.Mc, srcRunes[j])) {
			j++
		}
		// Copy the sequence into the target array
		for k := j - 1; k >= i; k-- {
			start--
			targetRunes[start] = srcRunes[k]
		}
		// Skip ahead the number of characters we processed
		i = j
	}
	return string(targetRunes[start:])
}
