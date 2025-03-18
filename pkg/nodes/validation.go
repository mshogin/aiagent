package nodes

import (
	"encoding/json"
	"fmt"
)

// ValidationNodeInterface defines the operations for a validation node
type ValidationNodeInterface interface {
	// Process validates the command output and determines if it meets the task goal
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// ValidationNode implements validation logic
type ValidationNode struct {
	llm           LLM
	ForceApproval bool
}

// NewValidationNode creates a new validation node
func NewValidationNode(llm LLM) *ValidationNode {
	return &ValidationNode{
		llm:           llm,
		ForceApproval: false,
	}
}

// Process implements the Node interface for ValidationNode
func (n *ValidationNode) Process(state *State) error {
	// Validate the command output
	prompt := fmt.Sprintf(`Validate the following command output:
Command: %s
Output: %s
Task Goal: %s

Return JSON response with:
{
    "is_valid": boolean,
    "issues": ["issue1", "issue2"],
    "explanation": "why the output is valid or not"
}`, state.Command, state.RawOutput, state.CurrentTask.Goal)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		IsValid     bool     `json:"is_valid"`
		Issues      []string `json:"issues"`
		Explanation string   `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return fmt.Errorf("failed to parse validation response: %v", err)
	}

	// Format validation result
	var output string
	if result.IsValid {
		output = fmt.Sprintf("✅ Validation passed: %s\n", result.Explanation)
	} else {
		output = fmt.Sprintf("❌ Validation failed: %s\n\nIssues:\n", result.Explanation)
		for _, issue := range result.Issues {
			output += fmt.Sprintf("- %s\n", issue)
		}
	}

	state.FinalResult = output
	state.NextNode = NodeTypeTerminal
	return nil
}

func (n *ValidationNode) Type() NodeType {
	return NodeTypeValidation
}
