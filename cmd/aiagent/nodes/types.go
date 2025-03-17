package nodes

// NodeType represents the type of a node in the langgraph
type NodeType string

const (
	// Core node types
	NodeTypeClassifier  NodeType = "classifier"
	NodeTypeBash        NodeType = "bash"
	NodeTypeValidation  NodeType = "validation"
	NodeTypeFormatter   NodeType = "formatter"
	NodeTypeTerminal    NodeType = "terminal"
	
	// Analytics node types
	NodeTypeContentCollection NodeType = "content_collection"
	NodeTypeAnalytics         NodeType = "analytics"
	NodeTypeDirectResponse    NodeType = "direct_response"
	NodeTypeCodeAnalyzer      NodeType = "code_analyzer"
)

// FileContent represents a file with its content
type FileContent struct {
	Path    string
	Content string
	Size    int64
	IsDir   bool
}

// State represents the shared state that is passed between nodes in the langgraph
type State struct {
	// Input is the original user input to the system
	Input string
	
	// Command is the bash command that has been generated
	Command string
	
	// NextNode determines which node should process the state next
	NextNode NodeType
	
	// FinalResult contains the final output to be returned to the user
	FinalResult string
	
	// RawOutput contains the unformatted command output before formatting
	RawOutput string
	
	// Verbose determines whether to show detailed processing information
	Verbose bool
	
	// WorkingDirectory contains the current working directory path
	WorkingDirectory string

	// AnalyticsFields contains fields used for analytics operations
	
	// DirectoryContents contains the list of files and directories found during content collection
	DirectoryContents []FileContent
	
	// NeedsFileContent determines if the analytics operation requires reading file contents
	NeedsFileContent bool
	
	// FilePatterns contains patterns of files to be read for analytics
	FilePatterns []string
	
	// FileCountLimit is the maximum number of files to read
	FileCountLimit int
	
	// FileSizeLimit is the maximum size (in bytes) of files to read
	FileSizeLimit int64
	
	// AnalyticsQuestion contains the specific analytical question to answer
	AnalyticsQuestion string
}

// Node represents a node in the langgraph
// Each node processes the current state and potentially updates it
type Node interface {
	// Process executes the node's logic on the provided state
	// It should update the state accordingly, including setting the NextNode field
	// to determine the next node in the workflow
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// LLM is an interface for language models
type LLM interface {
	// Generate sends a prompt to the language model and returns the generated text
	//
	// Parameters:
	//   - prompt: The input text prompt to send to the LLM
	//   - systemPrompt: Optional system prompt that helps set the behavior of the LLM
	//
	// Returns:
	//   - string: The generated text response from the LLM
	//   - error: An error if the generation fails
	Generate(prompt string, systemPrompt string) (string, error)
}