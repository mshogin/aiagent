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

	// Sanitize command
	if err := validateCommand(result.Command); err != nil {
		return "", fmt.Errorf("command validation failed: %v", err)
	}

	// Execute command
	cmd := exec.Command("bash", "-c", result.Command)
	cmd.Dir = state.WorkingDirectory // Set working directory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command execution failed: %v", err)
	}

	// Set result and next node
	state.CurrentTask.Result = strings.TrimSpace(string(output))
	state.NextNode = NodeTypeClassifier

	return state.CurrentTask.Result, nil
}

// validateCommand checks if a command is safe to execute
func validateCommand(cmd string) error {
	// List of dangerous commands/patterns
	dangerousPatterns := []string{
		"rm -rf",
		"rm -r",
		"sudo",
		">",
		">>",
		"|",
		"&",
		";",
		"`",
		"$(", // Command substitution
		"${", // Variable expansion
		"wget",
		"curl",
		"nc",
		"ncat",
		"telnet",
		"ftp",
		"ssh",
		"scp",
		"chmod",
		"chown",
		"chgrp",
		"mkfs",
		"dd",
		"mv",
		"cp",
	}

	cmdLower := strings.ToLower(cmd)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return fmt.Errorf("command contains dangerous pattern: %s", pattern)
		}
	}

	// Only allow specific safe commands
	allowedCommands := []string{
		"ls",
		"pwd",
		"echo",
		"cat",
		"head",
		"tail",
		"grep",
		"find",
		"df",
		"du",
		"free",
		"ps",
		"top",
		"uname",
		"whoami",
		"id",
		"date",
		"uptime",
		"hostname",
	}

	cmdParts := strings.Fields(cmd)
	if len(cmdParts) == 0 {
		return fmt.Errorf("empty command")
	}

	baseCmd := cmdParts[0]
	allowed := false
	for _, allowedCmd := range allowedCommands {
		if baseCmd == allowedCmd {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("command not in allowed list: %s", baseCmd)
	}

	return nil
}

func (n *BashNode) Type() NodeType {
	return NodeTypeBash
}
