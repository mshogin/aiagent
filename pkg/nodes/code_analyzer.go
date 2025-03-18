package nodes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CodeAnalyzerNodeInterface defines the operations for a code analyzer node
type CodeAnalyzerNodeInterface interface {
	// Process analyzes the codebase to find information about a specific subject
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// CodeAnalyzerNode implements code analysis logic
type CodeAnalyzerNode struct {
	llm LLM
}

// NewCodeAnalyzerNode creates a new code analyzer node
func NewCodeAnalyzerNode(llm LLM) *CodeAnalyzerNode {
	return &CodeAnalyzerNode{
		llm: llm,
	}
}

// Process implements the Node interface for CodeAnalyzerNode
func (n *CodeAnalyzerNode) Process(state *State) error {
	// Get file patterns to analyze
	needsContent, patterns, err := n.determineContentNeeds(state)
	if err != nil {
		return fmt.Errorf("failed to determine content needs: %v", err)
	}

	if !needsContent {
		return nil
	}

	// Find matching files
	files, err := n.findMatchingFiles(patterns)
	if err != nil {
		return fmt.Errorf("failed to find matching files: %v", err)
	}

	// Read file contents with safety checks
	contents := make(map[string]string)
	for _, file := range files {
		// Validate file path
		if err := validateFilePath(file, state.WorkingDirectory); err != nil {
			return fmt.Errorf("invalid file path: %v", err)
		}

		// Check file size
		info, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %v", file, err)
		}

		if info.Size() > state.FileSizeLimit {
			return fmt.Errorf("file %s exceeds size limit of %d bytes", file, state.FileSizeLimit)
		}

		// Read file with size limit
		content, err := readFileWithLimit(file, state.FileSizeLimit)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", file, err)
		}
		contents[file] = content
	}

	// Analyze contents
	analysis, err := n.analyzeContents(state, contents)
	if err != nil {
		return fmt.Errorf("failed to analyze contents: %v", err)
	}

	// Store the result
	state.FinalResult = analysis
	state.NextNode = NodeTypeTerminal

	return nil
}

func (n *CodeAnalyzerNode) determineContentNeeds(state *State) (bool, []string, error) {
	prompt := fmt.Sprintf(`Based on the current task, determine if code content analysis is needed:
Task Goal: %s
Working Directory: %s

Return JSON response with:
{
    "needs_content": boolean,
    "file_patterns": ["pattern1", "pattern2"],
    "explanation": "why content is needed or not"
}`, state.CurrentTask.Goal, state.WorkingDirectory)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return false, nil, fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		NeedsContent bool     `json:"needs_content"`
		FilePatterns []string `json:"file_patterns"`
		Explanation  string   `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return false, nil, fmt.Errorf("failed to parse content need response: %v", err)
	}

	return result.NeedsContent, result.FilePatterns, nil
}

func (n *CodeAnalyzerNode) findMatchingFiles(patterns []string) ([]string, error) {
	var matches []string
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob pattern %s: %v", pattern, err)
		}
		matches = append(matches, files...)
	}
	return matches, nil
}

func (n *CodeAnalyzerNode) analyzeContents(state *State, contents map[string]string) (string, error) {
	// Build content string
	var contentStr strings.Builder
	for file, content := range contents {
		contentStr.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", file, content))
	}

	prompt := fmt.Sprintf(`Analyze the following code contents based on the task goal:
Task Goal: %s

Code Contents:
%s

Return JSON response with:
{
    "analysis": "detailed analysis of the code",
    "recommendations": ["recommendation1", "recommendation2"],
    "explanation": "explanation of the analysis"
}`, state.CurrentTask.Goal, contentStr.String())

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return "", fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		Analysis        string   `json:"analysis"`
		Recommendations []string `json:"recommendations"`
		Explanation     string   `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", fmt.Errorf("failed to parse analysis response: %v", err)
	}

	return result.Analysis, nil
}

func (n *CodeAnalyzerNode) analyzeSubject(subject string, codeContext string, workingDir string) (string, error) {
	prompt := fmt.Sprintf(`Analyze the following code subject:
Subject: %s
Working Directory: %s

Code Context:
%s

Return JSON response with:
{
    "analysis": "detailed analysis of the subject",
    "recommendations": ["recommendation1", "recommendation2"],
    "explanation": "explanation of the analysis"
}`, subject, workingDir, codeContext)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return "", fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		Analysis        string   `json:"analysis"`
		Recommendations []string `json:"recommendations"`
		Explanation     string   `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", fmt.Errorf("failed to parse analysis response: %v", err)
	}

	return result.Analysis, nil
}

func (n *CodeAnalyzerNode) Type() NodeType {
	return NodeTypeCodeAnalyzer
}

// validateFilePath checks if a file path is safe to access
func validateFilePath(path string, workingDir string) error {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Convert working directory to absolute path
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute working directory: %v", err)
	}

	// Check if path is within working directory
	if !strings.HasPrefix(absPath, absWorkingDir) {
		return fmt.Errorf("file path %s is outside working directory %s", path, workingDir)
	}

	// Check for symlinks
	fileInfo, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks are not allowed: %s", path)
	}

	return nil
}

// readFileWithLimit reads a file with a size limit
func readFileWithLimit(path string, limit int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get file size
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %v", err)
	}

	// Check size limit
	if info.Size() > limit {
		return "", fmt.Errorf("file size %d exceeds limit %d", info.Size(), limit)
	}

	// Read file
	content := make([]byte, info.Size())
	_, err = file.Read(content)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	return string(content), nil
}
