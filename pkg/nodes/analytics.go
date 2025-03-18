package nodes

import (
	"encoding/json"
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
	llm LLM
}

// NewAnalyticsNode creates a new analytics node
func NewAnalyticsNode(llm LLM) *AnalyticsNode {
	return &AnalyticsNode{
		llm: llm,
	}
}

// Process implements the Node interface for AnalyticsNode
func (n *AnalyticsNode) Process(state *State) error {
	// Analyze task history and state
	prompt := fmt.Sprintf(`Analyze the task history and current state to provide insights:
Global Goal: %s
Task History: %v
Current State: %s

Return JSON response with:
{
    "insights": ["insight1", "insight2"],
    "recommendations": ["recommendation1", "recommendation2"],
    "explanation": "explanation of the analysis"
}`, state.GlobalGoal, state.TaskHistory, state.CurrentTask.Result)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		Insights        []string `json:"insights"`
		Recommendations []string `json:"recommendations"`
		Explanation     string   `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return fmt.Errorf("failed to parse analytics response: %v", err)
	}

	// Format insights
	var output string
	output += "Insights:\n"
	for _, insight := range result.Insights {
		output += fmt.Sprintf("- %s\n", insight)
	}
	output += "\nRecommendations:\n"
	for _, rec := range result.Recommendations {
		output += fmt.Sprintf("- %s\n", rec)
	}
	output += "\n" + result.Explanation

	state.RawOutput = output
	state.FinalResult = output

	// The analytics response should go directly to the terminal
	state.NextNode = NodeTypeTerminal

	return nil
}

func (n *AnalyticsNode) Type() NodeType {
	return NodeTypeAnalytics
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
