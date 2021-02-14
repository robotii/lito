package vm

// Pointer is used to point to an object.
// Variables should hold pointer instead of holding a object directly.
type Pointer struct {
	Target Object
}

type bits uint8

func (b bits) set(flag bits) bits    { return b | flag }
func (b bits) clear(flag bits) bits  { return b &^ flag }
func (b bits) toggle(flag bits) bits { return b ^ flag }
func (b bits) has(flag bits) bool    { return b&flag != 0 }

const (
	normal   = iota
	superRef
	namespace
)
