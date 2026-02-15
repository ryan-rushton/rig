package registry

import tea "github.com/charmbracelet/bubbletea"

// Tool defines a tool that can be launched from the home screen.
type Tool struct {
	ID          string
	Name        string
	Description string
	New         func() tea.Model
}

var tools []Tool

// Register adds a tool to the registry.
func Register(t Tool) {
	tools = append(tools, t)
}

// All returns all registered tools.
func All() []Tool {
	return tools
}

// Get returns the tool with the given ID, or nil if not found.
func Get(id string) *Tool {
	for i := range tools {
		if tools[i].ID == id {
			return &tools[i]
		}
	}
	return nil
}
