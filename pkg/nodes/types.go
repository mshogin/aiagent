package nodes

import (
	"fmt"
)

// NodeType represents the type of a node in the langgraph
type NodeType string

const (
	// Core node types
	NodeTypeClassifier NodeType = "classifier"
	NodeTypeBash       NodeType = "bash"
	NodeTypeValidation NodeType = "validation"
	NodeTypeFormatter  NodeType = "formatter"
	NodeTypeTerminal   NodeType = "terminal"

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

// TaskStatus represents the status of a task
type TaskStatus struct {
	NodeType    NodeType `json:"node_type"`
	Goal        string   `json:"goal"`
	IsCompleted bool     `json:"is_completed"`
	Result      string   `json:"result"`
}

// String returns a string representation of TaskStatus
func (t TaskStatus) String() string {
	return fmt.Sprintf("{NodeType:%s Goal:%s IsCompleted:%v Result:%s}", t.NodeType, t.Goal, t.IsCompleted, t.Result)
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

	// Task tracking fields
	CurrentTask TaskStatus   `json:"current_task"` // Current task being processed
	TaskHistory []TaskStatus `json:"task_history"` // History of completed tasks
	GlobalGoal  string       `json:"global_goal"`  // Overall goal to be achieved
	IsGoalMet   bool         `json:"is_goal_met"`  // Whether the global goal has been met

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
	//   - string: The result of the node's processing
	//   - error: An error if processing fails
	Process(state *State) (string, error)
	Type() NodeType
}
