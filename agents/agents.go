// Package agents handles agent parsing and configuration for orchestrate.
package agents

import (
	"strings"
)

// RGBColor represents an RGB color value.
type RGBColor struct {
	R, G, B int
}

// DefaultColors maps agent names to their display colors.
var DefaultColors = map[string]RGBColor{
	"droid":  {255, 140, 0},   // orange
	"claude": {210, 180, 140}, // sand/tan
	"codex":  {30, 30, 30},    // black
}

// Parse parses a comma-separated agents string into a slice.
// It handles optional brackets and trims whitespace from each agent.
func Parse(s string) []string {
	if s == "" {
		return nil
	}

	// Remove optional brackets
	s = strings.Trim(s, "[]")
	parts := strings.Split(s, ",")

	var agents []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			agents = append(agents, p)
		}
	}
	return agents
}

// GetColor returns the RGB color for an agent.
// Returns the color and true if found, zero RGBColor and false otherwise.
func GetColor(agent string) (RGBColor, bool) {
	color, ok := DefaultColors[agent]
	return color, ok
}

// Format formats a list of agents for display.
func Format(agents []string) string {
	return strings.Join(agents, ", ")
}

// Expand expands a list of agents by the given multiplier.
// Each agent in the list is repeated n times.
func Expand(agents []string, n int) []string {
	if n <= 0 {
		n = 1
	}
	result := make([]string, 0, len(agents)*n)
	for i := 0; i < n; i++ {
		result = append(result, agents...)
	}
	return result
}
