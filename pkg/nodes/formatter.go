package nodes

import (
	"encoding/json"
	"fmt"
)

// FormatterNodeInterface defines the operations for a formatter node
type FormatterNodeInterface interface {
	// Process formats the command output to make it more human-readable
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// FormatterNode implements the formatter node logic
type FormatterNode struct {
	llm LLM
}

// NewFormatterNode creates a new formatter node
func NewFormatterNode(llm LLM) *FormatterNode {
	return &FormatterNode{
		llm: llm,
	}
}

// Process implements the Node interface for FormatterNode
func (n *FormatterNode) Process(state *State) error {
	prompt := fmt.Sprintf(`Format the following output for better readability:
Raw Output: %s
Task Goal: %s

Return JSON response with:
{
    "formatted_output": "the formatted output",
    "explanation": "why this formatting was chosen"
}`, state.RawOutput, state.CurrentTask.Goal)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		FormattedOutput string `json:"formatted_output"`
		Explanation     string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return fmt.Errorf("failed to parse LLM response: %v", err)
	}

	state.NextNode = NodeTypeTerminal
	return nil
}

func (n *FormatterNode) Type() NodeType {
	return NodeTypeFormatter
}
