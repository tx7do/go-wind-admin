package hook

import (
	"fmt"
	"sort"
	"sync"
)

// Hook represents a hook point with attached scripts
type Hook struct {
	Name        string
	Description string
	Scripts     []*Script
}

// Script is imported from parent package to avoid circular dependency
type Script struct {
	ID          uint32
	Name        string
	Hook        string
	Source      string
	Enabled     bool
	Priority    int
	Description string
	Version     int
	Author      string
	Critical    bool
}

// Registry manages hooks and their scripts
type Registry struct {
	hooks map[string]*Hook
	mu    sync.RWMutex
}

// NewRegistry creates a new hook registry
func NewRegistry() *Registry {
	return &Registry{
		hooks: make(map[string]*Hook),
	}
}

// RegisterHook registers a new hook point
func (r *Registry) RegisterHook(name, description string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.hooks[name]; exists {
		return fmt.Errorf("hook already registered: %s", name)
	}

	r.hooks[name] = &Hook{
		Name:        name,
		Description: description,
		Scripts:     make([]*Script, 0),
	}

	return nil
}

// AddScript adds a script to a hook
func (r *Registry) AddScript(hookName string, script *Script) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hook, exists := r.hooks[hookName]
	if !exists {
		// Auto-register hook if it doesn't exist
		hook = &Hook{
			Name:    hookName,
			Scripts: make([]*Script, 0),
		}
		r.hooks[hookName] = hook
	}

	// Check for duplicate script names
	for _, s := range hook.Scripts {
		if s.Name == script.Name {
			return fmt.Errorf("script already exists: %s", script.Name)
		}
	}

	hook.Scripts = append(hook.Scripts, script)

	// Sort scripts by priority (lower priority = executed first)
	sort.Slice(hook.Scripts, func(i, j int) bool {
		return hook.Scripts[i].Priority < hook.Scripts[j].Priority
	})

	return nil
}

// RemoveScript removes a script from a hook
func (r *Registry) RemoveScript(hookName, scriptName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hook, exists := r.hooks[hookName]
	if !exists {
		return fmt.Errorf("hook not found: %s", hookName)
	}

	for i, script := range hook.Scripts {
		if script.Name == scriptName {
			hook.Scripts = append(hook.Scripts[:i], hook.Scripts[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("script not found: %s", scriptName)
}

// GetScripts returns all scripts for a hook
func (r *Registry) GetScripts(hookName string) []*Script {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hook, exists := r.hooks[hookName]
	if !exists {
		return []*Script{}
	}

	// Return copy to prevent external modification
	scripts := make([]*Script, len(hook.Scripts))
	copy(scripts, hook.Scripts)
	return scripts
}

// GetHook returns a hook by name
func (r *Registry) GetHook(name string) (*Hook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hook, exists := r.hooks[name]
	if !exists {
		return nil, fmt.Errorf("hook not found: %s", name)
	}

	return hook, nil
}

// ListHooks returns all registered hook names
func (r *Registry) ListHooks() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.hooks))
	for name := range r.hooks {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetAllHooks returns all hooks with their scripts
func (r *Registry) GetAllHooks() []*Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hooks := make([]*Hook, 0, len(r.hooks))
	for _, hook := range r.hooks {
		hooks = append(hooks, hook)
	}

	return hooks
}

// Clear removes all hooks and scripts
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hooks = make(map[string]*Hook)
}

// Count returns the total number of registered hooks
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.hooks)
}

// ScriptCount returns the total number of scripts across all hooks
func (r *Registry) ScriptCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, hook := range r.hooks {
		count += len(hook.Scripts)
	}
	return count
}
