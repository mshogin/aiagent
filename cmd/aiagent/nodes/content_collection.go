package nodes

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ContentCollectionNodeInterface defines the operations for a content collection node
type ContentCollectionNodeInterface interface {
	// Process collects directory content and optionally reads file contents
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// ContentCollectionNode implements content collection logic
type ContentCollectionNode struct {
	LLM     LLM
	Verbose bool
}

// NewContentCollectionNode creates a new content collection node
func NewContentCollectionNode(llm LLM, verbose bool) *ContentCollectionNode {
	return &ContentCollectionNode{
		LLM:     llm,
		Verbose: verbose,
	}
}

// Process implements the Node interface for ContentCollectionNode
func (n *ContentCollectionNode) Process(state *State) error {
	if n.Verbose {
		fmt.Println("Content collection node gathering information...")
		fmt.Printf("Working directory: %s\n", state.WorkingDirectory)
		if state.NeedsFileContent {
			fmt.Println("File content collection required")
			if len(state.FilePatterns) > 0 {
				fmt.Printf("File patterns: %v\n", state.FilePatterns)
			}
		} else {
			fmt.Println("Only collecting directory structure (no file contents)")
		}
	}

	// Set default limits if not provided
	if state.FileCountLimit <= 0 {
		state.FileCountLimit = 50 // Maximum number of files to read
	}
	if state.FileSizeLimit <= 0 {
		state.FileSizeLimit = 100 * 1024 // 100 KB maximum file size
	}

	// First, collect the directory structure
	dirContents, err := n.collectDirectoryContents(state.WorkingDirectory, state.FilePatterns, state.NeedsFileContent)
	if err != nil {
		return fmt.Errorf("failed to collect directory contents: %v", err)
	}

	state.DirectoryContents = dirContents

	if n.Verbose {
		fmt.Printf("Collected %d files/directories\n", len(state.DirectoryContents))
	}

	// Move to the analytics node next
	state.NextNode = NodeTypeAnalytics
	return nil
}

// collectDirectoryContents walks the directory tree and collects file information
func (n *ContentCollectionNode) collectDirectoryContents(rootDir string, patterns []string, readContents bool) ([]FileContent, error) {
	var contents []FileContent
	count := 0
	maxCount := 500 // Maximum number of files to track

	// Create a filepath.WalkDir function to collect directory contents
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip directories we can't access
		}

		// Early return if we've collected enough files
		if count >= maxCount {
			return filepath.SkipDir
		}

		// Skip hidden files and directories (starting with .)
		// But allow current working directory which might contain a leading "."
		if d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		isDir := d.IsDir()
		
		// Include all directories but only matching files if patterns are provided
		if !isDir && len(patterns) > 0 {
			matched := false
			for _, pattern := range patterns {
				match, err := filepath.Match(pattern, d.Name())
				if err != nil {
					continue // Invalid pattern
				}
				if match {
					matched = true
					break
				}
			}
			if !matched {
				return nil // Skip non-matching files
			}
		}

		// Skip very large files and binary files
		info, err := d.Info()
		if err != nil {
			return nil // Skip if we can't get file info
		}

		// Create FileContent object
		fileContent := FileContent{
			Path:  path,
			Size:  info.Size(),
			IsDir: isDir,
		}

		// Read file content if necessary and file is not too large
		if readContents && !isDir && info.Size() <= int64(100*1024) { // 100KB limit
			// Skip binary files and only read text files
			// This is a simple heuristic and might need improvement
			if !isTextFile(d.Name()) {
				fileContent.Content = "[binary file]"
			} else {
				content, err := os.ReadFile(path)
				if err != nil {
					fileContent.Content = fmt.Sprintf("[error reading file: %v]", err)
				} else {
					fileContent.Content = string(content)
				}
			}
		}

		contents = append(contents, fileContent)
		count++
		return nil
	})

	return contents, err
}

// isTextFile tries to determine if a file is a text file based on extension
func isTextFile(filename string) bool {
	textExtensions := []string{
		".txt", ".md", ".go", ".py", ".js", ".html", ".css", ".json", ".yaml", ".yml",
		".xml", ".sh", ".c", ".cpp", ".h", ".hpp", ".java", ".ts", ".tsx", ".jsx",
		".rb", ".rs", ".php", ".conf", ".cfg", ".ini", ".properties", ".toml", ".csv",
	}

	lowercaseName := strings.ToLower(filename)
	for _, ext := range textExtensions {
		if strings.HasSuffix(lowercaseName, ext) {
			return true
		}
	}

	return false
}