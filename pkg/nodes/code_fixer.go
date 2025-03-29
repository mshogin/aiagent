package nodes

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// CodeFixerNodeInterface defines the operations for a code fixer node
type CodeFixerNodeInterface interface {
	// Process analyzes the codebase, fixes issues, and runs tests
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// CodeFixerNode implements code fixing and testing logic
type CodeFixerNode struct {
	llm LLM
}

// NewCodeFixerNode creates a new code fixer node
func NewCodeFixerNode(llm LLM) *CodeFixerNode {
	return &CodeFixerNode{
		llm: llm,
	}
}

// Process implements the Node interface for CodeFixerNode
func (n *CodeFixerNode) Process(state *State) error {
	// First, analyze the current state and codebase
	analysis, err := n.analyzeCodebase(state)
	if err != nil {
		return fmt.Errorf("failed to analyze codebase: %v", err)
	}

	// Check if codebase is buildable
	if err := n.checkBuildability(state); err != nil {
		// If not buildable, try to fix the issues
		if err := n.fixBuildIssues(state, err.Error()); err != nil {
			return fmt.Errorf("failed to fix build issues: %v", err)
		}
	}

	// Run tests
	if err := n.runTests(state); err != nil {
		// If tests fail, try to fix the issues
		if err := n.fixTestIssues(state, err.Error()); err != nil {
			return fmt.Errorf("failed to fix test issues: %v", err)
		}
	}

	// Update the goal based on the analysis
	if err := n.updateGoal(state, analysis); err != nil {
		return fmt.Errorf("failed to update goal: %v", err)
	}

	// Build the new version
	if err := n.buildNewVersion(state); err != nil {
		return fmt.Errorf("failed to build new version: %v", err)
	}

	// Start the new version in detached mode
	if err := n.startNewVersion(state); err != nil {
		return fmt.Errorf("failed to start new version: %v", err)
	}

	// Exit the current process
	os.Exit(0)
	return nil
}

// analyzeCodebase analyzes the current codebase state
func (n *CodeFixerNode) analyzeCodebase(state *State) (string, error) {
	prompt := fmt.Sprintf(`Analyze the current codebase state:
Working Directory: %s
Current Goal: %s
Task History: %v

Return JSON response with:
{
    "issues": ["issue1", "issue2"],
    "suggestions": ["suggestion1", "suggestion2"],
    "next_steps": ["step1", "step2"],
    "analysis": "detailed analysis of the codebase"
}`, state.WorkingDirectory, state.GlobalGoal, state.TaskHistory)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return "", fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		Issues      []string `json:"issues"`
		Suggestions []string `json:"suggestions"`
		NextSteps   []string `json:"next_steps"`
		Analysis    string   `json:"analysis"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", fmt.Errorf("failed to parse analysis response: %v", err)
	}

	return result.Analysis, nil
}

// checkBuildability checks if the codebase can be built
func (n *CodeFixerNode) checkBuildability(state *State) error {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = state.WorkingDirectory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %s", string(output))
	}
	return nil
}

// runTests runs the test suite
func (n *CodeFixerNode) runTests(state *State) error {
	cmd := exec.Command("go", "test", "./...", "-v")
	cmd.Dir = state.WorkingDirectory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tests failed: %s", string(output))
	}
	return nil
}

// fixBuildIssues attempts to fix build issues
func (n *CodeFixerNode) fixBuildIssues(state *State, errorMsg string) error {
	prompt := fmt.Sprintf(`Fix the following build issues:
Error: %s
Working Directory: %s
Current Goal: %s

Return JSON response with:
{
    "fixes": ["fix1", "fix2"],
    "explanation": "explanation of the fixes",
    "files_to_modify": ["file1", "file2"]
}`, errorMsg, state.WorkingDirectory, state.GlobalGoal)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		Fixes         []string `json:"fixes"`
		Explanation   string   `json:"explanation"`
		FilesToModify []string `json:"files_to_modify"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return fmt.Errorf("failed to parse fix response: %v", err)
	}

	// Apply the fixes
	for _, file := range result.FilesToModify {
		if err := n.applyFix(file, result.Fixes); err != nil {
			return fmt.Errorf("failed to apply fix to %s: %v", file, err)
		}
	}

	return nil
}

// fixTestIssues attempts to fix test issues
func (n *CodeFixerNode) fixTestIssues(state *State, errorMsg string) error {
	prompt := fmt.Sprintf(`Fix the following test issues:
Error: %s
Working Directory: %s
Current Goal: %s

Return JSON response with:
{
    "fixes": ["fix1", "fix2"],
    "explanation": "explanation of the fixes",
    "files_to_modify": ["file1", "file2"]
}`, errorMsg, state.WorkingDirectory, state.GlobalGoal)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		Fixes         []string `json:"fixes"`
		Explanation   string   `json:"explanation"`
		FilesToModify []string `json:"files_to_modify"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return fmt.Errorf("failed to parse fix response: %v", err)
	}

	// Apply the fixes
	for _, file := range result.FilesToModify {
		if err := n.applyFix(file, result.Fixes); err != nil {
			return fmt.Errorf("failed to apply fix to %s: %v", file, err)
		}
	}

	return nil
}

// applyFix applies a fix to a file
func (n *CodeFixerNode) applyFix(file string, fixes []string) error {
	// Read the file
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Apply each fix
	for _, fix := range fixes {
		// Here you would implement the actual fix application logic
		// This could involve parsing the file, making changes, and writing back
		// For now, we'll just log the fix
		fmt.Printf("Applying fix to %s: %s\n", file, fix)
	}

	// Write the modified content back to the file
	if err := os.WriteFile(file, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// updateGoal updates the goal based on the analysis
func (n *CodeFixerNode) updateGoal(state *State, analysis string) error {
	prompt := fmt.Sprintf(`Based on the following analysis, determine the next goal:
Analysis: %s
Current Goal: %s
Task History: %v

Return JSON response with:
{
    "next_goal": "the next goal to achieve",
    "explanation": "why this goal was chosen"
}`, analysis, state.GlobalGoal, state.TaskHistory)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		NextGoal    string `json:"next_goal"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return fmt.Errorf("failed to parse goal response: %v", err)
	}

	// Update the global goal
	state.GlobalGoal = result.NextGoal
	return nil
}

// buildNewVersion builds the new version of the application
func (n *CodeFixerNode) buildNewVersion(state *State) error {
	// Build the application
	cmd := exec.Command("go", "build", "-o", "aiagent_new", "cmd/aiagent/main.go")
	cmd.Dir = state.WorkingDirectory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %s", string(output))
	}

	// Verify the new binary exists
	if _, err := os.Stat("aiagent_new"); err != nil {
		return fmt.Errorf("new binary not found: %v", err)
	}

	return nil
}

// startNewVersion starts the new version in detached mode
func (n *CodeFixerNode) startNewVersion(state *State) error {
	// Prepare command arguments
	args := []string{}
	if state.Verbose {
		args = append(args, "-v")
	}
	if state.GlobalGoal != "" {
		args = append(args, state.GlobalGoal)
	}

	// Create a new process group
	cmd := exec.Command("nohup", "./aiagent_new")
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = state.WorkingDirectory
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	// Redirect output to nohup.out
	outputFile, err := os.OpenFile("nohup.out", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %v", err)
	}
	defer outputFile.Close()

	cmd.Stdout = outputFile
	cmd.Stderr = outputFile

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start new version: %v", err)
	}

	// Detach the process
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to detach process: %v", err)
	}

	// Give the new process a moment to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (n *CodeFixerNode) Type() NodeType {
	return NodeTypeCodeFixer
}
