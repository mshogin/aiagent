package nodes

import (
	"fmt"
	"strings"
)

// AnalyticsNodeInterface defines the operations for an analytics node
type AnalyticsNodeInterface interface {
	// Process analyzes the collected file content and generates insights
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// AnalyticsNode implements the analytics node logic
type AnalyticsNode struct {
	LLM     LLM
	Verbose bool
}

// NewAnalyticsNode creates a new analytics node
func NewAnalyticsNode(llm LLM, verbose bool) *AnalyticsNode {
	return &AnalyticsNode{
		LLM:     llm,
		Verbose: verbose,
	}
}

// Process implements the Node interface for AnalyticsNode
func (n *AnalyticsNode) Process(state *State) error {
	if n.Verbose {
		fmt.Println("Analytics node processing directory content...")
		fmt.Printf("Analyzing %d files/directories\n", len(state.DirectoryContents))
		fmt.Printf("Question: %s\n", state.AnalyticsQuestion)
	}

	// Prepare a structured representation of the directory contents
	dirStructure, fileContents := n.prepareDirectoryInfo(state.DirectoryContents)

	systemPrompt := `You are an analytics system that analyzes directory contents and answers questions about them.
Your task is to analyze the provided directory structure and file contents to answer the user's question.

You should:
1. Analyze the directory structure to understand what files are available
2. Examine file contents when provided to extract relevant information
3. Generate a clear, concise answer to the user's question
4. Format your response with proper spacing and structure for readability
5. Use bullet points or sections when appropriate
6. When referencing files, use relative paths from the current directory
7. Respond only with the answer, no need for explanation of your analysis process

Remember: You're analyzing a file system, so consider file types, naming patterns, directory organization, 
and content when forming your response.`

	// Build prompt with directory structure and file contents
	prompt := fmt.Sprintf("Question: %s\n\nCurrent working directory: %s\n\n", 
		state.AnalyticsQuestion, state.WorkingDirectory)

	prompt += "Directory structure:\n" + dirStructure + "\n\n"

	if len(fileContents) > 0 {
		prompt += "File contents:\n" + fileContents
	}

	// Use LLM to analyze the data and answer the question
	response, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return fmt.Errorf("analytics LLM error: %v", err)
	}

	// Store the result and move to formatter
	state.RawOutput = response
	state.FinalResult = response

	// The analytics response should go directly to the terminal
	state.NextNode = NodeTypeTerminal
	
	return nil
}

// prepareDirectoryInfo formats directory information for the LLM
func (n *AnalyticsNode) prepareDirectoryInfo(contents []FileContent) (string, string) {
	var dirStructure strings.Builder
	var fileContents strings.Builder
	
	// Build a tree-like directory structure representation
	dirStructure.WriteString("```\n")
	for _, item := range contents {
		// Create relative path from working directory if possible
		path := item.Path
		if item.IsDir {
			path += "/"
		}
		dirStructure.WriteString(fmt.Sprintf("%s (%d bytes)\n", path, item.Size))
	}
	dirStructure.WriteString("```\n")
	
	// Include file contents when available (up to a reasonable limit)
	totalContentSize := 0
	maxContentSize := 100000 // Limit total content to ~100KB to avoid overwhelming the LLM
	
	for _, item := range contents {
		if !item.IsDir && len(item.Content) > 0 {
			// Skip if we've already included too much content
			if totalContentSize > maxContentSize {
				continue
			}
			
			// Truncate very large files
			content := item.Content
			if len(content) > 10000 {
				content = content[:10000] + "... [truncated]"
			}
			
			fileContents.WriteString(fmt.Sprintf("--- %s ---\n", item.Path))
			fileContents.WriteString(content)
			fileContents.WriteString("\n\n")
			
			totalContentSize += len(content)
		}
	}
	
	return dirStructure.String(), fileContents.String()
}