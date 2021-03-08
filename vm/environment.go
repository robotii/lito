package vm

import (
	"sort"
)

// Environment stores a set of Objects addressable by name
type Environment map[string]Object

func (e Environment) get(name string) (Object, bool) {
	obj, ok := e[name]
	return obj, ok
}

func (e Environment) set(name string, val Object) Object {
	e[name] = val
	return val
}

func (e Environment) names() []string {
	keys := make([]string, 0, len(e))
	for key := range e {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (e Environment) copy() Environment {
	newEnv := make(Environment, len(e))
	for key, value := range e {
		newEnv[key] = value
	}
	return newEnv
}

// EqualTo returns if the Environment is equal to another Environment
func (e Environment) EqualTo(other Environment) bool {
	if len(e) != len(other) {
		return false
	}
	for k, v := range e {
		if other[k] != v {
			return false
		}
	}
	return true
}
