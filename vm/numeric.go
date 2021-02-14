package vm

// Numeric represents a class that support numeric conversion to float.
type Numeric interface {
	floatValue() float64
	lessThan(object Object) bool
}
