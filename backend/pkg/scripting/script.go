package scripting

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// Script represents a Lua script
type Script struct {
	ID          uint32    `json:"id"`
	Name        string    `json:"name"`        // Unique script name
	Hook        string    `json:"hook"`        // Hook point name
	Source      string    `json:"source"`      // Lua source code
	Enabled     bool      `json:"enabled"`     // Active/inactive
	Priority    int       `json:"priority"`    // Execution order (lower = first)
	Description string    `json:"description"` // Human-readable description
	Version     int       `json:"version"`     // Version number
	Author      string    `json:"author"`      // Creator
	Critical    bool      `json:"critical"`    // If true, failure stops hook execution
	CreateTime  time.Time `json:"create_time"`
	UpdateTime  time.Time `json:"update_time"`
}

// Hash returns SHA256 hash of the script source
func (s *Script) Hash() string {
	h := sha256.New()
	h.Write([]byte(s.Source))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Validate validates the script
func (s *Script) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("script name is required")
	}
	if s.Hook == "" {
		return fmt.Errorf("hook name is required")
	}
	if s.Source == "" {
		return fmt.Errorf("script source is required")
	}
	return nil
}

// Clone creates a copy of the script
func (s *Script) Clone() *Script {
	return &Script{
		ID:          s.ID,
		Name:        s.Name,
		Hook:        s.Hook,
		Source:      s.Source,
		Enabled:     s.Enabled,
		Priority:    s.Priority,
		Description: s.Description,
		Version:     s.Version,
		Author:      s.Author,
		Critical:    s.Critical,
		CreateTime:  s.CreateTime,
		UpdateTime:  s.UpdateTime,
	}
}
