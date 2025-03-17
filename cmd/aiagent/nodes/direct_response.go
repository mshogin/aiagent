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
	LLM     LLM
	Verbose bool
}

// NewDirectResponseNode creates a new direct response node
func NewDirectResponseNode(llm LLM, verbose bool) *DirectResponseNode {
	return &DirectResponseNode{
		LLM:     llm,
		Verbose: verbose,
	}
}

// Process implements the Node interface for DirectResponseNode
func (n *DirectResponseNode) Process(state *State) error {
	if n.Verbose {
		fmt.Println("Direct response node processing question...")
	}

	systemPrompt := `You are an assistant that answers questions about directories and file systems.
Your task is to answer the user's question directly without executing any commands.

You should:
1. Use your knowledge about file systems and Linux/Unix commands to answer the question
2. Explain clearly how files, directories, and commands work
3. When appropriate, suggest commands that would help answer the question, but don't execute them
4. Be accurate, concise, and helpful in your explanations
5. If the question involves exploring or analyzing file content, explain that we need to use content analysis
6. Structure your answer clearly for easy reading

Focus on explaining how things work rather than executing commands to find answers.`

	prompt := fmt.Sprintf("Question about file system: %s\n\nCurrent working directory: %s", 
		state.Input, state.WorkingDirectory)

	// Generate the response using LLM
	response, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return fmt.Errorf("direct response LLM error: %v", err)
	}

	// Store the result and go to terminal
	state.FinalResult = response
	state.NextNode = NodeTypeTerminal

	return nil
}