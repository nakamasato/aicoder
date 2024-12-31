package llm

import "fmt"

// Node represents a node in the graph
type Node struct {
	ID    string
	Value string
}

// Graph represents the graph structure
type Graph struct {
	Nodes map[string]*Node
	Edges map[string][]string
}

// NewGraph creates a new graph
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make(map[string][]string),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(id, value string) {
	g.Nodes[id] = &Node{ID: id, Value: value}
}

// AddEdge adds an edge between two nodes
func (g *Graph) AddEdge(from, to string) {
	g.Edges[from] = append(g.Edges[from], to)
}

// Display prints the graph
func (g *Graph) Display() {
	for id, node := range g.Nodes {
		fmt.Printf("Node %s: %s\n", id, node.Value)
		for _, edge := range g.Edges[id] {
			fmt.Printf("  -> %s\n", edge)
		}
	}
}
