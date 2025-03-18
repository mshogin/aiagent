package nodes

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// BashNodeInterface defines the operations for a bash node
type BashNodeInterface interface {
	// Process analyzes the input and generates a bash command
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - string: The result of the command execution
	//   - error: An error if processing fails
	Process(state *State) (string, error)
}

// BashNode implements the bash command generation logic
type BashNode struct {
	llm LLM
}

// NewBashNode creates a new bash node
func NewBashNode(llm LLM) *BashNode {
	return &BashNode{
		llm: llm,
	}
}

// Process implements the Node interface for BashNode
func (n *BashNode) Process(state *State) (string, error) {
	// Get command from LLM
	prompt := fmt.Sprintf(`Based on the goal, generate a bash command to execute:
Goal: %s
Current State: %s

Return JSON response with:
{
    "command": "the bash command to execute",
    "explanation": "why this command was chosen"
}`, state.CurrentTask.Goal, state.Input)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to get command from LLM: %v", err)
	}

	// Parse response
	var result struct {
		Command     string `json:"command"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %v", err)
	}

	// Execute command
	cmd := exec.Command("bash", "-c", result.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command execution failed: %v", err)
	}

	// Set result and next node
	state.CurrentTask.Result = strings.TrimSpace(string(output))
	state.NextNode = NodeTypeClassifier

	return state.CurrentTask.Result, nil
}

func (n *BashNode) Type() NodeType {
	return NodeTypeBash
}
