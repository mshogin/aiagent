package nodes

import (
	"fmt"
	"strings"
)

// ClassifierNodeInterface defines the operations for a classifier node
type ClassifierNodeInterface interface {
	// Process analyzes the input and determines which node should process it next
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// ClassifierNode implements the classifier node logic
type ClassifierNode struct {
	LLM     LLM
	Verbose bool
}

// NewClassifierNode creates a new classifier node
func NewClassifierNode(llm LLM, verbose bool) *ClassifierNode {
	return &ClassifierNode{
		LLM:     llm,
		Verbose: verbose,
	}
}

// determineContentNeed analyzes the question to determine if file content needs to be read
// and which file patterns to include
func (n *ClassifierNode) determineContentNeed(question string, workingDir string) (bool, []string, error) {
	systemPrompt := `You are a file content analyzer that determines if a question requires reading file contents 
to answer properly, or if just having file names and directory structure is sufficient.

Respond in JSON format with two fields:
1. "needsContent": boolean (true if file contents are needed, false otherwise)
2. "filePatterns": array of strings (file patterns to match, e.g. ["*.go", "*.md"] or empty if not needed)

Examples:
- "What's the directory structure?" -> {"needsContent": false, "filePatterns": []}
- "List all files" -> {"needsContent": false, "filePatterns": []}
- "What programming languages are used in this project?" -> {"needsContent": true, "filePatterns": ["*.*"]}
- "Show me all Go files" -> {"needsContent": false, "filePatterns": ["*.go"]}
- "Analyze the code in this directory" -> {"needsContent": true, "filePatterns": ["*.go", "*.js", "*.py", "*.java", "*.c", "*.cpp", "*.h", "*.rb"]}
- "Summarize the markdown files" -> {"needsContent": true, "filePatterns": ["*.md", "*.markdown"]}

Only return valid JSON, nothing else.`

	prompt := fmt.Sprintf("Determine if this question requires reading file contents and which file patterns to include: \"%s\"\nWorking directory: %s", 
		question, workingDir)
	
	response, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return false, nil, fmt.Errorf("content need LLM error: %v", err)
	}
	
	// The response should be JSON, but let's be robust and extract the values
	responseLower := strings.ToLower(response)
	
	// Simple extraction for needsContent (this is a basic approach)
	needsContent := strings.Contains(responseLower, `"needscontent": true`) || 
					strings.Contains(responseLower, `"needsContent": true`)
	
	// Extract file patterns using basic string operations
	// This is a simplistic approach; in a production system you'd want proper JSON parsing
	var filePatterns []string
	
	if strings.Contains(responseLower, "filepatterns") {
		// Get everything after "filePatterns":
		split := strings.SplitN(responseLower, "filepatterns", 2)
		if len(split) > 1 {
			afterPatterns := split[1]
			// Find the array section
			start := strings.Index(afterPatterns, "[")
			end := strings.Index(afterPatterns, "]")
			if start != -1 && end != -1 && end > start {
				patternsSection := afterPatterns[start+1:end]
				// Split by commas and clean up
				rawPatterns := strings.Split(patternsSection, ",")
				for _, p := range rawPatterns {
					// Clean up the pattern
					pattern := strings.TrimSpace(p)
					pattern = strings.Trim(pattern, `"'`)
					if pattern != "" {
						filePatterns = append(filePatterns, pattern)
					}
				}
			}
		}
	}
	
	if n.Verbose {
		fmt.Printf("Content analysis: needsContent=%v, patterns=%v\n", 
			needsContent, filePatterns)
	}
	
	return needsContent, filePatterns, nil
}

// Process implements the Node interface for ClassifierNode
func (n *ClassifierNode) Process(state *State) error {
	if n.Verbose {
		fmt.Println("Classifier node analyzing input...")
	}
	
	// In real implementation, we would ask the LLM to classify the input
	systemPrompt := `You are a classifier that analyzes user inputs and determines which specialized node
should handle the request. There are now multiple possible nodes:

1. "bash" node: For tasks that require generating and executing bash commands
   Examples: "list files", "check disk space", "find text in files"

2. "content_collection" node: For analytical questions about the content of files in the directory
   Examples: "What file types are in this directory?", "Analyze the code structure", "Summarize the content of the files"
   NOTE: This node should be used when the question requires analyzing file CONTENT and not just directories/filenames.

3. "direct_response" node: For questions about file systems that don't need execution
   Examples: "What is a symbolic link?", "How does chmod work?", "Explain what the find command does"

4. "code_analyzer" node: For requesting detailed information about a specific component in the codebase
   Examples: "collect all information about the formatter", "explain how the validation system works", 
   "tell me about the parser class", "describe the authentication flow"
   NOTE: This node should be used when the user wants detailed information about a specific code component.

Your task is to output ONLY ONE of these node names: "bash", "content_collection", "direct_response", or "code_analyzer".

IMPORTANT: Keep in mind that the user is primarily interested in working with the CURRENT DIRECTORY. 
Their requests will likely be related to files, directories, and operations within their current
working directory. This context should be preserved as requests flow through the system.`

	prompt := fmt.Sprintf("Analyze this user request and determine which node should handle it: \"%s\"\nCurrent working directory: %s", 
		state.Input, state.WorkingDirectory)
	
	response, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return fmt.Errorf("classifier LLM error: %v", err)
	}
	
	// Parse the response to get the node type
	response = strings.ToLower(strings.TrimSpace(response))
	
	// Determine the next node based on the LLM's response
	if response == "content_collection" {
		// This is an analytics question, proceed with content collection
		if n.Verbose {
			fmt.Println("Classified as analytics question - proceeding to content collection")
		}
		
		// Prepare for content collection by setting the analytics question
		state.AnalyticsQuestion = state.Input
		
		// Now determine if we need to collect file contents or just directory structure
		needsContent, filePatterns, err := n.determineContentNeed(state.Input, state.WorkingDirectory)
		if err != nil {
			return fmt.Errorf("content need determination error: %v", err)
		}
		
		state.NeedsFileContent = needsContent
		state.FilePatterns = filePatterns
		state.FileCountLimit = 50  // Default to 50 files max
		state.FileSizeLimit = 100 * 1024  // Default to 100KB max file size
		
		state.NextNode = NodeTypeContentCollection
	} else if response == "direct_response" {
		// This is a question that can be answered directly without executing commands
		if n.Verbose {
			fmt.Println("Classified as direct response question")
		}
		state.NextNode = NodeTypeDirectResponse
	} else if response == "code_analyzer" {
		// This is a request for detailed information about a code component
		if n.Verbose {
			fmt.Println("Classified as code analysis request - analyzing specific component")
		}
		state.NextNode = NodeTypeCodeAnalyzer
	} else {
		// Default to bash command generation
		if n.Verbose {
			fmt.Println("Classified as bash command generation task")
		}
		state.NextNode = NodeTypeBash
	}
	
	if n.Verbose {
		fmt.Printf("Classifier determined the next node: %s\n", state.NextNode)
	}
	return nil
}