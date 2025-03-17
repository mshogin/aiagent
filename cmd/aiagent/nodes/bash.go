package nodes

import (
	"fmt"
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
	//   - error: An error if processing fails
	Process(state *State) error
}

// BashNode implements the bash command generation logic
type BashNode struct {
	LLM     LLM
	Verbose bool
}

// NewBashNode creates a new bash node
func NewBashNode(llm LLM, verbose bool) *BashNode {
	return &BashNode{
		LLM:     llm,
		Verbose: verbose,
	}
}

// Process implements the Node interface for BashNode
func (n *BashNode) Process(state *State) error {
	if n.Verbose {
		fmt.Println("Bash node generating command...")
	}
	
	systemPrompt := `You are a bash command generator. Your task is to convert user requests into effective bash commands.
Follow these guidelines:
1. Only respond with the bash command and nothing else
2. Prefer simple, standard commands when possible
3. Ensure the command is safe to execute
4. Do not include any explanation, just the command
5. For complex tasks, use appropriate flags and options
6. ALWAYS assume commands are being run in the CURRENT DIRECTORY, not root
7. Prioritize operations relative to the current directory
8. Use relative paths unless absolute paths are specifically requested

Examples:
- User asks to list files: respond with "ls -la"
- User asks about disk space: respond with "df -h ."
- User asks to find a file: respond with "find . -name filename 2>/dev/null"
- User asks for current directory info: respond with "pwd && ls -la"
- User asks to check directory size: respond with "du -sh ."
- User asks to search text in files: respond with "grep -r 'text' ."

IMPORTANT: Focus on the current directory context in all commands. For search operations, start from the current directory (.) rather than root (/)!`

	prompt := fmt.Sprintf("Generate a bash command to: %s\n\nCurrent working directory: %s", 
		state.Input, state.WorkingDirectory)
	
	response, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return fmt.Errorf("bash LLM error: %v", err)
	}
	
	// Clean up the response to get the command
	command := strings.TrimSpace(response)
	
	// Store the generated command in the state
	state.Command = command
	if n.Verbose {
		fmt.Printf("Generated bash command: %s\n", state.Command)
	}
	
	// Set the next node
	state.NextNode = NodeTypeValidation
	
	return nil
}