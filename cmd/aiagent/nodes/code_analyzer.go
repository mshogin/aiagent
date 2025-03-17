package nodes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	LLM     LLM
	Verbose bool
}

// NewCodeAnalyzerNode creates a new code analyzer node
func NewCodeAnalyzerNode(llm LLM, verbose bool) *CodeAnalyzerNode {
	return &CodeAnalyzerNode{
		LLM:     llm,
		Verbose: verbose,
	}
}

// Process implements the Node interface for CodeAnalyzerNode
func (n *CodeAnalyzerNode) Process(state *State) error {
	if n.Verbose {
		fmt.Println("Code analyzer node processing request...")
	}

	// Extract the subject from the input
	subject, err := n.extractSubject(state.Input)
	if err != nil {
		return fmt.Errorf("failed to extract subject: %v", err)
	}

	if n.Verbose {
		fmt.Printf("Analyzing subject: %s\n", subject)
	}

	// Collect relevant files and code snippets about the subject
	relevantFiles, err := n.findRelevantFiles(state.WorkingDirectory, subject)
	if err != nil {
		return fmt.Errorf("failed to find relevant files: %v", err)
	}

	if n.Verbose {
		fmt.Printf("Found %d relevant files\n", len(relevantFiles))
	}

	// If no files were found, try with a broader search
	if len(relevantFiles) == 0 {
		if n.Verbose {
			fmt.Printf("No files found with exact match, trying broader search...\n")
		}
		
		// Try to find related terms
		relatedTerms, err := n.findRelatedTerms(subject)
		if err != nil {
			return fmt.Errorf("failed to find related terms: %v", err)
		}
		
		// Search for files with related terms
		for _, term := range relatedTerms {
			termFiles, err := n.findRelevantFiles(state.WorkingDirectory, term)
			if err != nil {
				continue // Skip errors for related terms
			}
			relevantFiles = append(relevantFiles, termFiles...)
		}
		
		// Deduplicate files
		relevantFiles = n.deduplicateFiles(relevantFiles)
		
		if n.Verbose {
			fmt.Printf("After broader search: found %d relevant files\n", len(relevantFiles))
		}
	}

	// Collect code snippets and documentation from the relevant files
	codeContext, err := n.collectCodeContext(relevantFiles, subject)
	if err != nil {
		return fmt.Errorf("failed to collect code context: %v", err)
	}

	// Analyze the code context to generate a summary
	summary, err := n.analyzeSubject(subject, codeContext, state.WorkingDirectory)
	if err != nil {
		return fmt.Errorf("failed to analyze subject: %v", err)
	}

	// Store the result
	state.FinalResult = summary
	state.NextNode = NodeTypeTerminal

	return nil
}

// extractSubject extracts the subject to analyze from the user input
func (n *CodeAnalyzerNode) extractSubject(input string) (string, error) {
	// Use LLM to extract the subject from the input
	systemPrompt := `Extract the main technical subject or component name from the user's query. 
The subject is likely something that exists in a codebase, like a class, function, module, or concept.

For example:
- From "tell me about the Parser class" → "Parser"
- From "how does the validation system work?" → "validation system"
- From "collect all information about the formatter" → "formatter"
- From "explain the authentication flow" → "authentication flow"

Return ONLY the extracted subject without any explanation or additional text.`

	prompt := fmt.Sprintf("Extract the main technical subject from this query: %s", input)
	
	response, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to extract subject with LLM: %v", err)
	}
	
	// Clean up the response
	subject := strings.TrimSpace(response)
	
	// If we couldn't extract a clear subject, use a fallback approach
	if subject == "" {
		// Simple fallback: look for keywords like "about", "regarding", "on", etc.
		input = strings.ToLower(input)
		patterns := []string{
			`about the (\w+)`,
			`about (\w+)`,
			`regarding (\w+)`,
			`on (\w+)`,
			`how (\w+) works`,
		}
		
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(input)
			if len(matches) > 1 {
				subject = matches[1]
				break
			}
		}
	}
	
	if subject == "" {
		return "", fmt.Errorf("couldn't extract a clear subject from input")
	}
	
	return subject, nil
}

// findRelevantFiles searches for files that might contain information about the subject
func (n *CodeAnalyzerNode) findRelevantFiles(rootDir string, subject string) ([]string, error) {
	var relevantFiles []string
	subjectLower := strings.ToLower(subject)
	
	// Walk through the directory recursively
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}
		
		// Skip hidden directories (like .git)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			return filepath.SkipDir
		}
		
		// Skip non-code files and folders
		if info.IsDir() {
			return nil
		}
		
		// Skip binary or very large files
		if info.Size() > 1*1024*1024 { // Skip files larger than 1MB
			return nil
		}
		
		// Check if file extension is relevant for code
		ext := strings.ToLower(filepath.Ext(path))
		if !isCodeFile(ext) {
			return nil
		}
		
		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}
		
		// Check if subject appears in filename or content
		fileName := strings.ToLower(filepath.Base(path))
		contentLower := strings.ToLower(string(content))
		
		if strings.Contains(fileName, subjectLower) || strings.Contains(contentLower, subjectLower) {
			relevantFiles = append(relevantFiles, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("error walking through directory: %v", err)
	}
	
	return relevantFiles, nil
}

// findRelatedTerms uses the LLM to generate related terms for the subject
func (n *CodeAnalyzerNode) findRelatedTerms(subject string) ([]string, error) {
	systemPrompt := `Generate 3-5 related technical terms or variations for the given subject that might appear in code.
For example, if the subject is "formatter", related terms might include "format", "formatting", "pretty", "output formatting", etc.
Return the terms as a comma-separated list without any other text.`

	prompt := fmt.Sprintf("Generate related technical terms for: %s", subject)
	
	response, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate related terms: %v", err)
	}
	
	// Parse the comma-separated list
	terms := strings.Split(response, ",")
	var cleanTerms []string
	
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term != "" && term != subject {
			cleanTerms = append(cleanTerms, term)
		}
	}
	
	return cleanTerms, nil
}

// deduplicateFiles removes duplicate file paths from the list
func (n *CodeAnalyzerNode) deduplicateFiles(files []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, file := range files {
		if !seen[file] {
			seen[file] = true
			result = append(result, file)
		}
	}
	
	return result
}

// collectCodeContext extracts relevant code snippets and documentation from files
func (n *CodeAnalyzerNode) collectCodeContext(files []string, subject string) (string, error) {
	var contextBuilder strings.Builder
	subjectPattern := regexp.MustCompile(fmt.Sprintf(`(?i)(\b%s\b|%s)`, regexp.QuoteMeta(subject), regexp.QuoteMeta(subject)))
	
	for _, file := range files {
		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files we can't read
		}
		
		contentStr := string(content)
		
		// Extract relevant snippets
		snippets := n.extractRelevantSnippets(contentStr, subjectPattern, file)
		
		if snippets != "" {
			contextBuilder.WriteString(fmt.Sprintf("File: %s\n", file))
			contextBuilder.WriteString(snippets)
			contextBuilder.WriteString("\n\n")
		}
	}
	
	return contextBuilder.String(), nil
}

// extractRelevantSnippets extracts code snippets containing the subject
func (n *CodeAnalyzerNode) extractRelevantSnippets(content string, subjectPattern *regexp.Regexp, filePath string) string {
	var snippetsBuilder strings.Builder
	lines := strings.Split(content, "\n")
	
	// Determine file type based on extension
	ext := strings.ToLower(filepath.Ext(filePath))
	
	// Extract language-specific elements
	if ext == ".go" {
		// For Go files, extract types, functions, and methods related to the subject
		n.extractGoElements(lines, subjectPattern, &snippetsBuilder)
	} else {
		// Generic extraction for other file types
		n.extractGenericElements(lines, subjectPattern, &snippetsBuilder)
	}
	
	return snippetsBuilder.String()
}

// extractGoElements extracts relevant Go code elements
func (n *CodeAnalyzerNode) extractGoElements(lines []string, subjectPattern *regexp.Regexp, sb *strings.Builder) {
	var inRelevantBlock bool
	var blockBuffer []string
	
	typePattern := regexp.MustCompile(`^\s*(type|func|var|const)\s+(\w+)`)
	
	for i, line := range lines {
		// Check for type, function, or method declarations
		if matches := typePattern.FindStringSubmatch(line); matches != nil {
			// Found a new declaration, process the previous block if needed
			if inRelevantBlock && len(blockBuffer) > 0 {
				sb.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", strings.Join(blockBuffer, "\n")))
				blockBuffer = []string{}
			}
			
			// Check if this declaration contains the subject
			if subjectPattern.MatchString(line) {
				inRelevantBlock = true
				blockBuffer = append(blockBuffer, line)
				continue
			} else {
				inRelevantBlock = false
			}
		}
		
		// If we're in a relevant block, add lines to the buffer
		if inRelevantBlock {
			blockBuffer = append(blockBuffer, line)
			
			// Check if this might be the end of a block (closing brace at start of line)
			if strings.TrimSpace(line) == "}" {
				// If we have enough context, add this block
				if len(blockBuffer) > 0 {
					sb.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", strings.Join(blockBuffer, "\n")))
					blockBuffer = []string{}
					inRelevantBlock = false
				}
			}
		} else {
			// If not in a block but line contains subject, add a snippet with context
			if subjectPattern.MatchString(line) {
				startLine := max(0, i-3)
				endLine := min(len(lines), i+4)
				
				snippetLines := lines[startLine:endLine]
				sb.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", strings.Join(snippetLines, "\n")))
			}
		}
	}
	
	// Add any remaining block
	if inRelevantBlock && len(blockBuffer) > 0 {
		sb.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", strings.Join(blockBuffer, "\n")))
	}
}

// extractGenericElements extracts relevant code elements for non-Go files
func (n *CodeAnalyzerNode) extractGenericElements(lines []string, subjectPattern *regexp.Regexp, sb *strings.Builder) {
	for i, line := range lines {
		if subjectPattern.MatchString(line) {
			// Extract a few lines before and after for context
			startLine := max(0, i-3)
			endLine := min(len(lines), i+4)
			
			sb.WriteString("```\n")
			for j := startLine; j < endLine; j++ {
				sb.WriteString(lines[j] + "\n")
			}
			sb.WriteString("```\n\n")
		}
	}
}

// analyzeSubject uses the LLM to analyze the collected code context
func (n *CodeAnalyzerNode) analyzeSubject(subject string, codeContext string, workingDir string) (string, error) {
	systemPrompt := `You are a code analysis expert. Your task is to provide a concise but comprehensive analysis of a subject in a codebase.

Analyze the provided code snippets and produce a technical summary that includes:
1. What the subject is (class, function, system, concept)
2. Its purpose and main responsibilities
3. How it's implemented
4. How it interacts with other components
5. Any important design patterns or architectural decisions
6. Usage examples or important APIs

Format your analysis as follows:
- Start with a brief description of what the subject is
- Use sections with headings for different aspects (Purpose, Implementation, Interfaces, etc.)
- Include code examples where relevant
- Provide clear explanations of technical concepts
- Use markdown formatting for readability

Keep your response focused on the technical details. Be concise but thorough.`

	// If we have a lot of code context, we need to summarize it first
	var contextToUse string
	if len(codeContext) > 5000 {
		contextToUse = n.summarizeContext(codeContext, subject)
	} else {
		contextToUse = codeContext
	}

	prompt := fmt.Sprintf("Subject to analyze: %s\nWorking directory: %s\n\nCode context:\n%s", 
		subject, workingDir, contextToUse)
	
	analysis, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to analyze subject: %v", err)
	}
	
	return analysis, nil
}

// summarizeContext summarizes a large code context to fit within token limits
func (n *CodeAnalyzerNode) summarizeContext(codeContext string, subject string) string {
	parts := strings.Split(codeContext, "File: ")
	
	var summaryBuilder strings.Builder
	
	// Start with a system message
	summaryBuilder.WriteString("Note: The code context has been summarized due to size.\n\n")
	
	// Try to include the most relevant files directly
	subjectLower := strings.ToLower(subject)
	
	// First pass: Include files that mention the subject in the file name
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		lines := strings.SplitN(part, "\n", 2)
		if len(lines) < 2 {
			continue
		}
		
		filePath := lines[0]
		content := lines[1]
		
		if strings.Contains(strings.ToLower(filePath), subjectLower) {
			summaryBuilder.WriteString("File: " + filePath + "\n")
			summaryBuilder.WriteString(content + "\n\n")
		}
	}
	
	// Second pass: If we don't have enough material, add other relevant files
	if summaryBuilder.Len() < 1000 {
		for _, part := range parts {
			if part == "" {
				continue
			}
			
			lines := strings.SplitN(part, "\n", 2)
			if len(lines) < 2 {
				continue
			}
			
			filePath := lines[0]
			content := lines[1]
			
			// Skip files we've already included
			if strings.Contains(summaryBuilder.String(), "File: "+filePath) {
				continue
			}
			
			// Add the file content up to a reasonable size
			if summaryBuilder.Len()+len(content) < 5000 {
				summaryBuilder.WriteString("File: " + filePath + "\n")
				summaryBuilder.WriteString(content + "\n\n")
			}
		}
	}
	
	return summaryBuilder.String()
}

// isCodeFile checks if a file extension is a code file
func isCodeFile(ext string) bool {
	codeExtensions := map[string]bool{
		".go":   true,
		".js":   true,
		".ts":   true,
		".py":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".rb":   true,
		".php":  true,
		".cs":   true,
		".rs":   true,
		".swift": true,
		".kt":   true,
		".scala": true,
		".sh":   true,
		".bash": true,
		".pl":   true,
		".ex":   true,
		".ml":   true,
	}
	
	return codeExtensions[ext]
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}