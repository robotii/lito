package bytecode

// localTable holds mappings from variable names to the index of the local
type localTable struct {
	store map[string]int
	count int
	depth int
	upper *localTable
}

func (lt *localTable) get(v string) (int, bool) {
	i, ok := lt.store[v]
	return i, ok
}

func (lt *localTable) set(val string) int {
	c, ok := lt.store[val]

	if !ok {
		c = lt.count
		lt.store[val] = c
		lt.count++
	}

	return c
}

func (lt *localTable) setLocal(v string, d int) (index, depth int) {
	index, depth, ok := lt.getLocal(v, d)

	if !ok {
		index = lt.set(v)
		depth = 0
	}

	return index, depth
}

func (lt *localTable) getLocal(v string, d int) (index, depth int, ok bool) {
	index, ok = lt.get(v)
	if ok {
		return index, d - lt.depth, ok
	}
	if lt.upper != nil {
		return lt.upper.getLocal(v, d)
	}
	return -1, 0, false
}

func newLocalTable(depth int) *localTable {
	return &localTable{store: make(map[string]int), depth: depth}
}
