package fsm

type (
	// FSM represents a finite state machine with a current state and list of
	// valid state transitions.
	FSM struct {
		current     string
		transitions map[stateKey]bool
	}

	// State represents a signel state and the states that are allowed to
	// transition to this state.
	State struct {
		Name string
		From []string
	}

	// stateKey holds the source and destination for a state transition.
	stateKey struct {
		event string
		src   string
	}
)

// States a list of states with the valid transitions
type States []State

// New creates a new state machine, with the initial state
// and state transitions provided.
func New(initial string, states States) *FSM {
	f := &FSM{
		current:     initial,
		transitions: make(map[stateKey]bool),
	}

	for _, state := range states {
		for _, src := range state.From {
			f.transitions[stateKey{state.Name, src}] = true
		}
	}

	return f
}

// State sets the current state to that provided, assuming that the
// state transition is valid.
func (f *FSM) State(state string) {
	if f.transitions[stateKey{state, f.current}] {
		f.current = state
	}
}

// Is returns if the machine is in the state provided.
func (f *FSM) Is(state string) bool {
	return f.current == state
}

// Current returns the current state of the machine
func (f *FSM) Current() string {
	return f.current
}
