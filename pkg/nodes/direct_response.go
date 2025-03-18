package nodes

import (
	"fmt"
)

// DirectResponseNodeInterface defines the operations for a direct response node
type DirectResponseNodeInterface interface {
	// Process generates a direct response to the user's question without executing any commands
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// DirectResponseNode implements direct response logic
type DirectResponseNode struct {
	llm LLM
}

// NewDirectResponseNode creates a new direct response node
func NewDirectResponseNode(llm LLM) *DirectResponseNode {
	return &DirectResponseNode{
		llm: llm,
	}
}

// Process implements the Node interface for DirectResponseNode
func (n *DirectResponseNode) Process(state *State) error {
	prompt := fmt.Sprintf(`Based on the current task, provide a direct response:
Task Goal: %s
Current State: %s`, state.CurrentTask.Goal, state.Input)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return fmt.Errorf("LLM error: %v", err)
	}

	state.FinalResult = response
	state.NextNode = NodeTypeTerminal
	return nil
}

func (n *DirectResponseNode) Type() NodeType {
	return NodeTypeDirectResponse
}
