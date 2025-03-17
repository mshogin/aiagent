package nodes

import (
	"fmt"
	"strings"
)

// FormatterNodeInterface defines the operations for a formatter node
type FormatterNodeInterface interface {
	// Process formats the command output to make it more human-readable
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// FormatterNode implements the formatter node logic
type FormatterNode struct {
	LLM     LLM
	Verbose bool
}

// NewFormatterNode creates a new formatter node
func NewFormatterNode(llm LLM, verbose bool) *FormatterNode {
	return &FormatterNode{
		LLM:     llm,
		Verbose: verbose,
	}
}

// Process implements the Node interface for FormatterNode
func (n *FormatterNode) Process(state *State) error {
	if n.Verbose {
		fmt.Println("Formatter node processing output...")
	}

	// If there's no output to format, just skip this node
	if state.FinalResult == "" || strings.Contains(state.FinalResult, "cancelled") {
		if n.Verbose {
			fmt.Println("No output to format, skipping formatter")
		}
		state.NextNode = NodeTypeTerminal
		return nil
	}

	// Use the raw output if available, otherwise extract from FinalResult
	var commandOutput string
	if state.RawOutput != "" {
		commandOutput = state.RawOutput
	} else {
		commandOutput = state.FinalResult
		if strings.HasPrefix(commandOutput, "Command output:\n") {
			commandOutput = strings.TrimPrefix(commandOutput, "Command output:\n")
		}
	}

	systemPrompt := `You are a terminal output formatter. Your task is to format command line output to make it more human-readable 
and visually appealing. Follow these guidelines:

1. Use colors, spacing, and formatting to make the output easier to understand
2. Organize tabular data in a clean way
3. Highlight important information (like status, sizes, permissions, etc.)
4. Add helpful explanations where appropriate
5. For command outputs that include tables, use proper table formatting
6. NEVER add explanatory text at the beginning or end, just format the output
7. IMPORTANT: Use \x1b instead of \033 for your escape sequences to ensure compatibility

IMPORTANT: When output contains Markdown formatting:
- Convert markdown headings to colored and bold headings 
- Convert markdown code blocks (triple backticks) to bordered and syntax-highlighted blocks
- Convert markdown lists to properly indented and styled lists
- Convert markdown emphasis (asterisk or underscore wrapped text) to italic text
- Convert markdown strong emphasis (double asterisks or underscores) to bold text
- Convert markdown links to underlined colored text
- Maintain inline code formatting with appropriate coloring

IMPORTANT: Remember that commands are being executed in the user's CURRENT DIRECTORY.
- Highlight current directory path in pwd output
- For file listings, emphasize the current directory (.) and parent directory (..)
- When formatting search results, make paths relative to current directory more noticeable
- Highlight any information that gives context about the current directory

Some useful ANSI color codes you can use:
- Bold: \x1b[1m
- Underline: \x1b[4m
- Italic: \x1b[3m
- Red: \x1b[31m
- Green: \x1b[32m
- Yellow: \x1b[33m
- Blue: \x1b[34m
- Magenta: \x1b[35m
- Cyan: \x1b[36m
- White: \x1b[37m
- Reset: \x1b[0m
- Background Blue: \x1b[44m
- Background Grey: \x1b[48;5;240m

Examples of what to improve:
- Format ls -la output with colors based on file types and permissions, highlighting current directory
- Organize df -h output into a proper table with colors for usage percentages, emphasizing the current directory mount
- When formatting find/grep results, make local files stand out from system files
- For directory size output (du), highlight the current directory size
- In file content output (cat/grep), highlight line numbers and path information
- For markdown content, convert headings, code blocks, and formatting to terminal-friendly equivalents`

	prompt := fmt.Sprintf("Format this terminal output from the command '%s' to make it more readable:\n\nCommand was executed in directory: %s\n\nOutput:\n%s",
		state.Command, state.WorkingDirectory, commandOutput)

	formattedOutput, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return fmt.Errorf("formatter LLM error: %v", err)
	}

	formattedOutput = strings.ReplaceAll(formattedOutput, "\\x1b", "\x1b")

	// Update the final result with the formatted output
	state.FinalResult = formattedOutput

	// Set the next node
	state.NextNode = NodeTypeTerminal

	return nil
}
