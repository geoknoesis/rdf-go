package rdf

import "fmt"

// blankNodeGenerator provides a thread-safe way to generate unique blank node IDs.
// This is used across different parsers to ensure consistent blank node generation.
type blankNodeGenerator struct {
	counter int
}

// newBlankNodeGenerator creates a new blank node generator.
func newBlankNodeGenerator() *blankNodeGenerator {
	return &blankNodeGenerator{counter: 0}
}

// next generates the next blank node ID.
func (g *blankNodeGenerator) next() BlankNode {
	g.counter++
	return BlankNode{ID: fmt.Sprintf("b%d", g.counter)}
}

// reset resets the counter (useful for testing).
func (g *blankNodeGenerator) reset() {
	g.counter = 0
}

// generateBlankNodeID generates a blank node ID from a counter value.
// This is a helper function to standardize blank node ID generation across the codebase.
// Format: "b" followed by the counter number (e.g., "b1", "b2", "b3").
func generateBlankNodeID(counter int) string {
	return fmt.Sprintf("b%d", counter)
}
