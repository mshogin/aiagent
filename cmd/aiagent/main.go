package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"aiagent/pkg/nodes"
)

func main() {
	// Define configuration flags
	useMock := flag.Bool("mock", false, "Use mock LLM instead of real API")
	verbose := flag.Bool("v", false, "Enable verbose mode (show detailed processing information)")
	forceApprove := flag.Bool("y", false, "Auto-approve commands without validation (use with caution)")
	flag.Parse()

	// Get input from CLI arguments (combine all args into a single string)
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Error: Please provide an input argument")
		fmt.Println("Usage: aiagent [--mock] [-v] [-y] your request here")
		fmt.Println("  --mock         Use mock LLM instead of real API")
		fmt.Println("  -v             Enable verbose mode (show detailed processing information)")
		fmt.Println("  -y             Auto-approve commands without validation (use with caution)")
		os.Exit(1)
	}

	// Validate and sanitize input
	input, err := validateAndSanitizeInput(args)
	if err != nil {
		fmt.Printf("Error: Invalid input: %v\n", err)
		os.Exit(1)
	}

	// Only show verbose output if -v flag is used
	if *verbose {
		fmt.Printf("Received input: %s\n", input)
		if *forceApprove {
			fmt.Println("Warning: Force approval mode enabled. Commands will execute without validation.")
		}
	}

	// Choose LLM implementation based on flag
	var llm nodes.LLM
	if *useMock {
		if *verbose {
			fmt.Println("Using mock LLM")
		}
		llm = &MockLLM{}
	} else {
		if *verbose {
			fmt.Println("Using real LLM API")
		}
		llm = nodes.NewDefaultLLM()
	}

	// Initialize and run the langgraph
	result, err := runLangGraph(input, llm, *verbose, *forceApprove)
	if err != nil {
		fmt.Printf("Error running langgraph: %v\n", err)
		os.Exit(1)
	}

	// Print the final result without any prefix
	fmt.Print(result)
}

// validateAndSanitizeInput validates and sanitizes user input
func validateAndSanitizeInput(args []string) (string, error) {
	// Join arguments
	input := strings.Join(args, " ")

	// Check input length
	if len(input) > 1000 {
		return "", fmt.Errorf("input too long (max 1000 characters)")
	}

	// Check for dangerous patterns
	dangerousPatterns := []string{
		"../",
		"./",
		"~",
		"$(",
		"`",
		"${",
		"&&",
		"||",
		";",
		"|",
		">",
		"<",
		"\\",
		"\"",
		"'",
		"\x00", // Null byte
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(input, pattern) {
			return "", fmt.Errorf("input contains dangerous pattern: %s", pattern)
		}
	}

	// Only allow printable ASCII characters
	for _, r := range input {
		if r < 32 || r > 126 {
			return "", fmt.Errorf("input contains invalid character: %q", r)
		}
	}

	return input, nil
}

// MockLLM implements a simple mock LLM for testing the system
type MockLLM struct{}

func (m *MockLLM) Complete(prompt string) (string, error) {
	promptLower := strings.ToLower(prompt)

	// Check if this is the formatter output phase (after code_analyzer has already been selected)
	if strings.Contains(promptLower, "subject to analyze") {
		// Return markdown content that will be formatted
		return "# Formatter Component Analysis\n\n" +
			"## Overview\n" +
			"The Formatter is a component in the aiagent system responsible for formatting command output to improve readability. " +
			"It takes raw command output and applies formatting enhancements such as syntax highlighting, color coding, and structural organization.\n\n" +
			"## Implementation\n" +
			"The Formatter is implemented as a node in the langgraph architecture.\n\n" +
			"```go\n" +
			"type FormatterNode struct {\n" +
			"    LLM     LLM\n" +
			"    Verbose bool\n" +
			"}\n" +
			"```\n\n" +
			"## Key Features\n" +
			"1. **ANSI Color Support**: Uses terminal escape sequences for highlighting\n" +
			"2. **Context-Aware Formatting**: Adapts formatting based on command type\n" +
			"3. **Special Handlers**: Custom formatters for common commands\n" +
			"4. **Directory Highlighting**: Emphasizes the current directory in output\n\n" +
			"## Integration Points\n" +
			"- Receives input from the ValidationNode after command execution\n" +
			"- Final node in the processing pipeline before terminal output\n" +
			"- Communicates with State object to access raw command output\n\n" +
			"## Usage\n" +
			"The formatter is automatically invoked as part of the langgraph pipeline.", nil
	}

	// Manually process this very specific test case to show code analyzer output
	if strings.Contains(promptLower, "collect all information about the formatter") ||
		strings.Contains(promptLower, "formatter component") ||
		strings.Contains(promptLower, "tell me about formatter") {
		return "code_analyzer", nil
	}

	// Otherwise, delegate to the real mock implementation
	return nodes.DefaultMockGenerate(prompt, "")
}

// runLangGraph orchestrates the flow between nodes
func runLangGraph(input string, llm nodes.LLM, verbose bool, forceApprove bool) (string, error) {
	// Create core nodes
	classifierNode := nodes.NewClassifierNode(llm)
	bashNode := nodes.NewBashNode(llm)
	validationNode := nodes.NewValidationNode(llm)
	validationNode.ForceApproval = forceApprove // Set force approval flag
	formatterNode := nodes.NewFormatterNode(llm)

	// Create analytics nodes
	contentCollectionNode := nodes.NewContentCollectionNode(llm, verbose)
	analyticsNode := nodes.NewAnalyticsNode(llm)
	directResponseNode := nodes.NewDirectResponseNode(llm)
	codeAnalyzerNode := nodes.NewCodeAnalyzerNode(llm)
	codeFixerNode := nodes.NewCodeFixerNode(llm)

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}

	if verbose {
		fmt.Printf("Working in directory: %s\n", cwd)
	}

	// Create initial state
	state := &nodes.State{
		Input:            input,
		NextNode:         nodes.NodeTypeClassifier,
		Verbose:          verbose,
		WorkingDirectory: cwd,
		FileCountLimit:   50,     // Default to max 50 files
		FileSizeLimit:    100000, // Default to max 100KB per file
		GlobalGoal:       input,  // Set the original input as the global goal
		TaskHistory:      make([]nodes.TaskStatus, 0),
	}

	// Run the graph until we reach a terminal state
	for state.NextNode != nodes.NodeTypeTerminal {
		var err error
		var result string

		switch state.NextNode {
		// Core nodes
		case nodes.NodeTypeClassifier:
			result, err = classifierNode.Process(state)
		case nodes.NodeTypeBash:
			result, err = bashNode.Process(state)
			state.CurrentTask.Result = result
			state.NextNode = nodes.NodeTypeClassifier // Route back to classifier
		case nodes.NodeTypeValidation:
			err = validationNode.Process(state)
			state.CurrentTask.Result = state.RawOutput
			state.NextNode = nodes.NodeTypeClassifier // Route back to classifier
		case nodes.NodeTypeFormatter:
			err = formatterNode.Process(state)
			state.CurrentTask.Result = state.RawOutput
			state.NextNode = nodes.NodeTypeClassifier // Route back to classifier

		// Analytics nodes
		case nodes.NodeTypeContentCollection:
			err = contentCollectionNode.Process(state)
			state.CurrentTask.Result = state.RawOutput
			state.NextNode = nodes.NodeTypeClassifier // Route back to classifier
		case nodes.NodeTypeAnalytics:
			err = analyticsNode.Process(state)
			state.CurrentTask.Result = state.RawOutput
			state.NextNode = nodes.NodeTypeClassifier // Route back to classifier
		case nodes.NodeTypeDirectResponse:
			err = directResponseNode.Process(state)
			state.CurrentTask.Result = state.RawOutput
			state.NextNode = nodes.NodeTypeClassifier // Route back to classifier
		case nodes.NodeTypeCodeAnalyzer:
			err = codeAnalyzerNode.Process(state)
			state.CurrentTask.Result = state.RawOutput
			state.NextNode = nodes.NodeTypeClassifier // Route back to classifier
		case nodes.NodeTypeCodeFixer:
			err = codeFixerNode.Process(state)
			state.CurrentTask.Result = state.RawOutput
			state.NextNode = nodes.NodeTypeClassifier // Route back to classifier

		default:
			return "", fmt.Errorf("invalid node type: %s", state.NextNode)
		}

		if err != nil {
			return "", fmt.Errorf("error in node %s: %v", state.NextNode, err)
		}

		// Update FinalResult with the latest result if available
		if result != "" {
			state.FinalResult = result
		}
	}

	return state.FinalResult, nil
}
