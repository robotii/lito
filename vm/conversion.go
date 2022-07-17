package vm

import "os"

// InitObjectFromGoType returns an object that can be used from Lito
func (vm *VM) InitObjectFromGoType(value interface{}) Object {
	switch val := value.(type) {
	case nil:
		return NIL

	case int:
		return IntegerObject(val)

	case int64:
		return IntegerObject(int(val))

	case int32:
		return IntegerObject(int(val))

	case float64:
		return FloatObject(val)

	case []uint8: // also handles []byte
		bytes := make([]byte, len(val))
		copy(bytes, val)
		return StringObject(bytes)

	case string:
		return StringObject(val)

	case bool:
		return BooleanObject(val)

	case []interface{}:
		objects := make([]Object, len(val))
		for i, elem := range val {
			objects[i] = vm.InitObjectFromGoType(elem)
		}
		return InitArrayObject(objects)

	case map[string]interface{}:
		pairs := map[string]Object{}
		for k, value := range val {
			pairs[k] = vm.InitObjectFromGoType(value)
		}
		return InitHashObject(pairs)

	case *os.File:
		return initFileObject(vm, val)

	case os.File:
		return initFileObject(vm, &val)

	default:
		if val == nil {
			return NIL
		}
		o, ok := val.(Object)
		if ok {
			return o
		}
		return initGoObject(vm, value)
	}
}

// InitGoTypeFromObject returns an object that can be used from Lito
func (vm *VM) InitGoTypeFromObject(value Object) interface{} {
	switch val := value.(type) {
	case *NilObject:
		return nil

	case IntegerObject:
		return int(val)

	case FloatObject:
		return float64(val)

	case StringObject:
		return string(val)

	case BooleanObject:
		return bool(val)

	case *ArrayObject:
		a := make([]interface{}, len(val.Elements))
		for i, v := range val.Elements {
			a[i] = vm.InitGoTypeFromObject(v)
		}
		return a

	case *HashObject:
		m := make(map[string]interface{})
		for k, value := range val.Pairs {
			m[k] = vm.InitGoTypeFromObject(value)
		}
		return m

	case *FileObject:
		return val.File

	default:
		return val
	}
}
