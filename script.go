package ankiddie

// Script contains dynamic behavior for the system to implement at run time
type Script struct {
	ID      string
	Type    string
	Autorun int
	Code    string
	Notes   string
}

// ScriptLoader loads scripts from storage
type ScriptLoader interface {
	// GetScript should return an error if a script with the specified ID does not exist
	GetScript(id string) (*Script, error)
	GetAutorunScripts(autorunLevel int) ([]*Script, error)
}

// ScriptStorer stores scripts to storage
type ScriptStorer interface {
	StoreScript(script *Script) error
}

// ScriptPersister loads and stores scripts from storage
type ScriptPersister interface {
	ScriptLoader
	ScriptStorer
}
