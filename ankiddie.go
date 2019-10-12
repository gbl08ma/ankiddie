package ankiddie

import (
	"errors"
	"reflect"
	"sync"

	uuid "github.com/satori/go.uuid"

	"github.com/gbl08ma/anko/env"
)

const scriptType = "anko"

// ErrNoPersister when a script persister is not specified
var ErrNoPersister = errors.New("script persister not specified")

// Ankiddie manages the execution of anko scripts
type Ankiddie struct {
	m     sync.Mutex
	envs  map[uint]*Environment
	curID uint
	store ScriptPersister
}

// PackageConfigurator configures additional packages to expose to anko environments
type PackageConfigurator interface {
	ConfigurePackages(packages map[string]map[string]reflect.Value, packageTypes map[string]map[string]reflect.Type)
}

// New returns a new Ankiddie
func New(configurator PackageConfigurator, store ScriptPersister) *Ankiddie {
	ankiddie := &Ankiddie{
		envs:  make(map[uint]*Environment),
		store: store,
	}

	// TODO because of how anko works, this will actually mess with the packages
	// for all envs on all Ankiddies, and not just this one
	if configurator != nil {
		configurator.ConfigurePackages(env.Packages, env.PackageTypes)
	}
	return ankiddie
}

// NewEnvWithCode returns a new Environment ready to run the provided code
func (ssys *Ankiddie) NewEnvWithCode(code string, out func(env *Environment, msg string) error) *Environment {
	ssys.m.Lock()
	defer ssys.m.Unlock()
	env := ssys.newEnv(ssys.curID, code, out)
	ssys.envs[env.eid] = env
	ssys.curID++
	return env
}

// NewEnvWithScript returns a new Environment ready to run the provided Script
func (ssys *Ankiddie) NewEnvWithScript(script *Script, out func(env *Environment, msg string) error) *Environment {
	ssys.m.Lock()
	defer ssys.m.Unlock()
	env := ssys.newEnv(ssys.curID, script.Code, out)
	env.scriptID = script.ID
	ssys.envs[env.eid] = env
	ssys.curID++
	return env
}

// Environment returns the environment with the given ID, if one exists
func (ssys *Ankiddie) Environment(eid uint) (*Environment, bool) {
	ssys.m.Lock()
	defer ssys.m.Unlock()
	env, ok := ssys.envs[eid]
	return env, ok
}

// Environments returns a map with the currently registered environments
func (ssys *Ankiddie) Environments() map[uint]*Environment {
	ssys.m.Lock()
	defer ssys.m.Unlock()
	envscopy := make(map[uint]*Environment)
	for key, env := range ssys.envs {
		envscopy[key] = env
	}
	return envscopy
}

// ForgetEnv stops execution of the given environment as far as possible and unregisters it
func (ssys *Ankiddie) ForgetEnv(env *Environment) {
	ssys.m.Lock()
	defer ssys.m.Unlock()
	env.cancel()
	delete(ssys.envs, env.eid)
}

// FullReset stops execution on all environments and destroys them
func (ssys *Ankiddie) FullReset() {
	ssys.m.Lock()
	defer ssys.m.Unlock()
	for _, env := range ssys.envs {
		env.cancel()
	}
	ssys.envs = make(map[uint]*Environment)
}

// StartAutorun executes scripts at the specified autorun level
func (ssys *Ankiddie) StartAutorun(level int, async bool, out func(env *Environment, msg string) error) error {
	if ssys.store == nil {
		return ErrNoPersister
	}

	scripts, err := ssys.store.GetAutorunScripts(level)
	if err != nil {
		return err
	}

	for _, script := range scripts {
		env := ssys.NewEnvWithScript(script, out)
		if async {
			go env.Start()
		} else {
			env.Start()
		}
	}
	return nil
}

// SaveScript saves a script to the database under the specified ID
// If no ID is provided, a UUID is generated
// If a script with the same ID already existed, it is overwritten
func (ssys *Ankiddie) SaveScript(id string, code string) (*Script, error) {
	if ssys.store == nil {
		return nil, ErrNoPersister
	}

	if id == "" {
		uid, err := uuid.NewV4()
		if err != nil {
			return nil, err
		}
		id = uid.String()
	}

	script, err := ssys.store.GetScript(id)
	if err != nil {
		script = &Script{
			ID:      id,
			Autorun: -1,
		}
	}

	script.Type = scriptType
	script.Code = code

	return script, ssys.store.StoreScript(script)
}
