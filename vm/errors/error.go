package errors

const (
	// Error is the root of the error classes. All errors should inherit from this.
	Error = "Error"
	// InternalError is the default error type
	InternalError = "InternalError"
	// IOError is an IO error such as file error
	IOError = "IOError"
	// ArgumentError is for an argument-related error
	ArgumentError = "ArgumentError"
	// NameError is for a constant-related error
	NameError = "NameError"
	// TypeError is for a type-related error
	TypeError = "TypeError"
	// NoMethodError is for an intentionally unsupported-method error
	NoMethodError = "NoMethodError"
	// ConstantAlreadyInitialisedError means user re-declares twice
	ConstantAlreadyInitialisedError = "ConstantAlreadyInitialisedError"
	// ZeroDivisionError is for zero-division by Integer/Float value
	ZeroDivisionError = "ZeroDivisionError"
	// ChannelCloseError is for accessing to the closed channel
	ChannelCloseError = "ChannelCloseError"
	// NotImplementedError is for features that have not been implemented
	NotImplementedError = "NotImplementedError"
)

//	Here defines different error message formats for different types of errors
const (
	WrongNumberOfArgument       = "Expect %d argument(s). got: %d"
	WrongNumberOfArgumentMore   = "Expect %d or more argument(s). got: %d"
	WrongNumberOfArgumentLess   = "Expect %d or less argument(s). got: %d"
	WrongNumberOfArgumentRange  = "Expect %d to %d argument(s). got: %d"
	WrongArgumentTypeFormat     = "Expect argument to be %s. got: %s"
	WrongArgumentTypeFormatNum  = "Expect argument #%d to be %s. got: %s"
	InvalidChmodNumber          = "Invalid chmod number. got: %d"
	InvalidNumericString        = "Invalid numeric string. got: %s"
	CantLoadFile                = "Can't load \"%s\""
	CantRequireNonString        = "Can't require \"%s\": Pass a string instead"
	CantYieldWithoutBlockFormat = "Can't yield without a block"
	DividedByZero               = "Divided by 0"
	ChannelIsClosed             = "The channel is already closed."
	TooSmallIndexValue          = "Index value %d too small for array. minimum: %d"
	IndexOutOfRange             = "Index value out of range. got: %v"
	NegativeValue               = "Expect argument to be positive value. got: %d"
	NegativeSecondValue         = "Expect second argument to be positive value. got: %d"
	UndefinedMethod             = "Undefined Method '%+v' for %+v"
)

// Classes a list of error classes to be initialised
var Classes = [...]string{
	InternalError,
	IOError,
	ArgumentError,
	NameError,
	TypeError,
	NoMethodError,
	ConstantAlreadyInitialisedError,
	ZeroDivisionError,
	ChannelCloseError,
	NotImplementedError,
}
