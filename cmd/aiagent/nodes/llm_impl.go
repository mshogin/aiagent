package nodes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// DefaultLLM implements the LLM interface using a simple API call
type DefaultLLM struct {
	ApiUrl    string
	ApiKey    string
	ModelId   string
	MaxTokens int
}

// ChatMessage represents a message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a request to the chat completion API
type ChatCompletionRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

// ChatCompletionResponse represents the response from the chat completion API
type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewDefaultLLM creates a new instance of DefaultLLM
func NewDefaultLLM() *DefaultLLM {
	// Default to OpenAI API, but can be replaced with any compatible API
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: OPENAI_API_KEY environment variable not set")
	}

	return &DefaultLLM{
		ApiUrl:    "https://api.openai.com/v1/chat/completions",
		ApiKey:    apiKey,
		ModelId:   "gpt-3.5-turbo",
		MaxTokens: 1000,
	}
}

// Generate implements the LLM interface for DefaultLLM
func (llm *DefaultLLM) Generate(prompt string, systemPrompt string) (string, error) {
	if llm.ApiKey == "" {
		return "", errors.New("API key not set. Please set the OPENAI_API_KEY environment variable")
	}

	messages := []ChatMessage{}
	
	// Add system prompt if provided
	if systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	
	// Add user prompt
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	requestBody := ChatCompletionRequest{
		Model:     llm.ModelId,
		Messages:  messages,
		MaxTokens: llm.MaxTokens,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", llm.ApiUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llm.ApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		errorMsg := "unknown error"
		if result.Error.Message != "" {
			errorMsg = result.Error.Message
		}
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, errorMsg)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// MockLLM implements the LLM interface for testing purposes
type MockLLM struct{}

// Generate implements the LLM interface
func (llm *MockLLM) Generate(prompt string, systemPrompt string) (string, error) {
	return DefaultMockGenerate(prompt, systemPrompt)
}

// DefaultMockGenerate provides a default mock implementation of the Generate method
func DefaultMockGenerate(prompt string, systemPrompt string) (string, error) {
	// For testing, return simple canned responses based on prompts
	promptLower := strings.ToLower(prompt)

	// Handle classifier node requests
	if strings.Contains(promptLower, "which node") || strings.Contains(promptLower, "classifier") {
		// Check for analytics related terms
		if strings.Contains(promptLower, "analyze") || strings.Contains(promptLower, "code") || 
		   strings.Contains(promptLower, "files") || strings.Contains(promptLower, "content") || 
		   strings.Contains(promptLower, "summarize") || strings.Contains(promptLower, "what type") {
			return "content_collection", nil
		}
		
		// Check for specific component analysis terms
		if strings.Contains(promptLower, "collect all information about") || 
		   strings.Contains(promptLower, "explain how") || strings.Contains(promptLower, "tell me about") ||
		   strings.Contains(promptLower, "describe the") {
			return "code_analyzer", nil
		}
		
		// Check for direct response terms
		if strings.Contains(promptLower, "explain") || strings.Contains(promptLower, "what is") || 
		   strings.Contains(promptLower, "how does") || strings.Contains(promptLower, "tell me about") {
			return "direct_response", nil
		}
		
		// Default to bash commands
		return "bash", nil
	}

	// Handle content need determination
	if strings.Contains(promptLower, "requires reading file contents") || strings.Contains(promptLower, "file patterns") {
		if strings.Contains(promptLower, "structure") || strings.Contains(promptLower, "list") {
			return `{"needsContent": false, "filePatterns": []}`, nil
		} else if strings.Contains(promptLower, "code") || strings.Contains(promptLower, "analyze") {
			return `{"needsContent": true, "filePatterns": ["*.go", "*.js", "*.py", "*.java", "*.c", "*.cpp"]}`, nil
		} else if strings.Contains(promptLower, "markdown") || strings.Contains(promptLower, "documentation") {
			return `{"needsContent": true, "filePatterns": ["*.md", "*.txt", "README*", "CHANGELOG*"]}`, nil
		} else {
			return `{"needsContent": true, "filePatterns": ["*.*"]}`, nil
		}
	}
	
	// Handle bash node requests
	if strings.Contains(promptLower, "generate") || strings.Contains(promptLower, "bash command") {
		if strings.Contains(promptLower, "list") && strings.Contains(promptLower, "file") {
			return "ls -la", nil
		}
		if strings.Contains(promptLower, "directory") {
			return "pwd", nil
		}
		if strings.Contains(promptLower, "disk") {
			return "df -h", nil
		}
		if strings.Contains(promptLower, "memory") {
			return "free -h", nil
		}
		if strings.Contains(promptLower, "system") {
			return "uname -a", nil
		}
		return "echo 'Hello world'", nil
	}

	// Handle validation node requests
	if strings.Contains(promptLower, "safe") || strings.Contains(promptLower, "validation") || strings.Contains(promptLower, "analyze") {
		// For dangerous commands
		if strings.Contains(promptLower, "rm -rf") || strings.Contains(promptLower, "sudo") {
			return "DANGEROUS [8] This command has high potential for system damage as it involves destructive operations that could permanently delete data or modify system settings.", nil
		}
		
		// For commands that need caution
		if strings.Contains(promptLower, "mv") || strings.Contains(promptLower, "cp") || 
		   strings.Contains(promptLower, ">") || strings.Contains(promptLower, "chmod") {
			return "CAUTION [5] This command modifies files or permissions but is generally safe when used correctly. Verify the target paths before execution.", nil
		}
		
		// For standard safe commands
		return "SAFE [2] This command is safe to execute. It only reads information without modifying any system files or settings.", nil
	}
	
	// Handle alternative command suggestions
	if strings.Contains(promptLower, "suggest an alternative command") {
		// Check for common errors and suggest fixes
		if strings.Contains(promptLower, "command not found") || strings.Contains(promptLower, "not installed") {
			if strings.Contains(promptLower, "pip ") {
				return "pip3 install package-name", nil
			} else if strings.Contains(promptLower, "python ") {
				return "python3 script.py", nil
			} else if strings.Contains(promptLower, "node ") {
				return "nodejs script.js", nil
			} else if strings.Contains(promptLower, "gcc ") {
				return "apt-get install build-essential", nil
			} else {
				return "which command-name", nil
			}
		}
		
		// For file not found errors
		if strings.Contains(promptLower, "no such file") || strings.Contains(promptLower, "cannot find") {
			if strings.Contains(promptLower, "ls ") {
				return "ls -la .", nil
			} else if strings.Contains(promptLower, "cat ") {
				return "find . -name \"*file*\" -type f", nil
			} else if strings.Contains(promptLower, "cd ") {
				return "ls -la && mkdir -p directory && cd directory", nil
			} else {
				return "find . -type f | grep -i filename", nil
			}
		}
		
		// For permission errors
		if strings.Contains(promptLower, "permission denied") {
			if strings.Contains(promptLower, "chmod ") {
				return "ls -la file && sudo chmod +x file", nil
			} else {
				return "sudo bash -c \"original command\"", nil
			}
		}
		
		// Default alternative
		return "ls -la && pwd", nil
	}
	
	// Handle code analyzer requests
	if strings.Contains(promptLower, "extract the main technical subject") {
		if strings.Contains(promptLower, "formatter") {
			return "formatter", nil
		} else if strings.Contains(promptLower, "validation") {
			return "validation", nil
		} else if strings.Contains(promptLower, "bash") {
			return "bash command", nil
		} else if strings.Contains(promptLower, "analytics") {
			return "analytics system", nil
		} else {
			return "code component", nil
		}
	}
	
	if strings.Contains(promptLower, "generate related technical terms") {
		if strings.Contains(promptLower, "formatter") {
			return "format, formatting, output formatting, pretty print, display", nil
		} else if strings.Contains(promptLower, "validation") {
			return "validate, validator, verification, safety check, command validation", nil
		} else if strings.Contains(promptLower, "bash") {
			return "shell, command, terminal, execution, command line", nil
		} else {
			return "component, module, system, function, class", nil
		}
	}
	
	if strings.Contains(promptLower, "subject to analyze") {
		if strings.Contains(promptLower, "formatter") {
			return "# Formatter Component Analysis\n\n" +
				"## Overview\n" +
				"The Formatter is a component in the aiagent system responsible for formatting command output to improve readability. " +
				"It takes raw command output and applies formatting enhancements such as syntax highlighting, color coding, and structural organization.\n\n" +
				"## Implementation\n" +
				"The Formatter is implemented as a node in the langgraph architecture.\n\n" +
				"```go\n" +
				"type FormatterNode struct {\n" +
				"    LLM     LLM\n" +
				"    Verbose bool\n" +
				"}\n" +
				"```\n\n" +
				"## Key Features\n" +
				"1. **ANSI Color Support**: Uses terminal escape sequences for highlighting\n" +
				"2. **Context-Aware Formatting**: Adapts formatting based on command type\n" +
				"3. **Special Handlers**: Custom formatters for common commands\n" +
				"4. **Directory Highlighting**: Emphasizes the current directory in output\n\n" +
				"## Integration Points\n" +
				"- Receives input from the ValidationNode after command execution\n" +
				"- Final node in the processing pipeline before terminal output\n" +
				"- Communicates with State object to access raw command output\n\n" +
				"## Usage\n" +
				"The formatter is automatically invoked as part of the langgraph pipeline.", nil
		} else if strings.Contains(promptLower, "validation") {
			return "# Validation Component Analysis\n\n" +
				"## Overview\n" +
				"The Validation component is responsible for assessing bash commands for safety, obtaining user confirmation when needed, " +
				"and executing approved commands.\n\n" +
				"## Implementation\n" +
				"Located in validation.go, the ValidationNode implements several key features.\n\n" +
				"```go\n" +
				"func (n *ValidationNode) validateWithLLM(command string) (string, int, error) {\n" +
				"    // LLM-based validation logic\n" +
				"    // Returns safety rating and explanation\n" +
				"}\n" +
				"```\n\n" +
				"## Security Features\n" +
				"1. **Pattern Matching**: Checks for known dangerous commands\n" +
				"2. **LLM Assessment**: Uses AI to evaluate command safety\n" +
				"3. **User Confirmation**: Requires explicit approval for risky operations\n" +
				"4. **Working Directory Context**: Considers risks to the current directory\n\n" +
				"## Integration Points\n" +
				"- Receives commands from BashNode\n" +
				"- Passes output to FormatterNode\n" +
				"- Can recursively process alternative commands\n\n" +
				"## Usage\n" +
				"The validation node automatically processes all commands generated by the system.", nil
		} else {
			return "# Code Component Analysis\n\n" +
				"## Overview\n" +
				"The requested component is part of the aiagent system, which is a CLI tool built with Go that processes user requests.\n\n" +
				"## Implementation\n" +
				"The component is implemented in Go using a node-based architecture pattern.\n\n" +
				"```go\n" +
				"// Generic node interface\n" +
				"type Node interface {\n" +
				"    Process(state *State) error\n" +
				"}\n" +
				"```\n\n" +
				"## Key Features\n" +
				"1. **Modular Design**: Separate nodes for different responsibilities\n" +
				"2. **LLM Integration**: Uses language models for processing\n" +
				"3. **Safety Features**: Command validation and risk assessment\n" +
				"4. **Rich Formatting**: Terminal-friendly colorized output\n\n" +
				"## Integration Points\n" +
				"The system follows a sequential processing flow through multiple nodes.\n\n" +
				"## Usage\n" +
				"The component is used as part of the aiagent CLI tool with various configuration options.", nil
		}
	}

	// Handle markdown formatting specifically
	if strings.Contains(promptLower, "format") && (strings.Contains(promptLower, "markdown") || 
	   strings.Contains(promptLower, "#") || strings.Contains(promptLower, "code block")) {
		// For markdown output
		return "\x1b[1m\x1b[34m# Markdown Heading Level 1\x1b[0m\n\n" +
			"\x1b[1m\x1b[36m## Heading Level 2\x1b[0m\n\n" +
			"Regular paragraph text with \x1b[3mitalic emphasis\x1b[0m and \x1b[1mbold text\x1b[0m.\n\n" +
			"\x1b[33m* First list item\x1b[0m\n" +
			"\x1b[33m* Second list item\x1b[0m\n" +
			"\x1b[33m  * Nested list item\x1b[0m\n\n" +
			"\x1b[48;5;240m\x1b[37mCode block (Go):\n" +
			"func Example() string {\n" +
			"    fmt.Println(\"This is formatted as code\")\n" +
			"    return \"success\"\n" +
			"}\x1b[0m\n\n" +
			"Text with \x1b[32minline code\x1b[0m formatting.\n\n" +
			"\x1b[4m\x1b[34mhttps://example.com\x1b[0m - a link", nil
	}

	// Handle formatter node requests
	if strings.Contains(promptLower, "format") || strings.Contains(promptLower, "formatter") {
		if strings.Contains(promptLower, "df -h") {
			// For df -h commands, return a colorized table
			return "\x1b[1m\x1b[36mFilesystem      Size  Used Avail Use% Mounted on\x1b[0m\n" +
				"\x1b[32m/dev/sda1       \x1b[0m 50G   25G   25G  \x1b[31m50%\x1b[0m  /\n" +
				"\x1b[32m/dev/sdb1       \x1b[0m 100G  20G   80G  \x1b[32m20%\x1b[0m  /home\n" +
				"\x1b[32m/dev/sdc1       \x1b[0m 200G  180G  20G  \x1b[31m90%\x1b[0m  /data", nil
		}
		if strings.Contains(promptLower, "ls -la") {
			// For ls -la commands, return a colorized listing
			return "total 32\n" +
				"\x1b[34mdrwxr-xr-x  5 user user 4096 Mar 17 10:00 \x1b[1m\x1b[34m.\x1b[0m\n" +
				"\x1b[34mdrwxr-xr-x 20 user user 4096 Mar 17 09:50 \x1b[1m\x1b[34m..\x1b[0m\n" +
				"-rw-r--r--  1 user user 1250 Mar 17 09:55 \x1b[32mREADME.md\x1b[0m\n" +
				"\x1b[34mdrwxr-xr-x  3 user user 4096 Mar 17 09:52 \x1b[1m\x1b[34mcmd\x1b[0m\n" +
				"-rw-r--r--  1 user user  492 Mar 17 09:51 \x1b[32mgo.mod\x1b[0m\n" +
				"-rw-r--r--  1 user user  740 Mar 17 09:51 \x1b[32mgo.sum\x1b[0m\n" +
				"\x1b[32m-rwxr-xr-x  1 user user 8192 Mar 17 10:00 \x1b[1m\x1b[32maiagent\x1b[0m", nil
		}
		if strings.Contains(promptLower, "uname -a") {
			// For uname -a commands, return a colorized system info
			return "\x1b[1m\x1b[35mLinux\x1b[0m \x1b[32mhostname\x1b[0m \x1b[33m5.15.0-76-generic\x1b[0m #83-Ubuntu SMP " +
				"Wed Feb 14 10:54:05 UTC 2024 \x1b[36mx86_64\x1b[0m GNU/Linux", nil
		}
		if strings.Contains(promptLower, "free -h") {
			// For free -h commands, return a colorized memory info
			return "\x1b[1m\x1b[36m              total        used        free      shared  buff/cache   available\x1b[0m\n" +
				"\x1b[1mMem:          \x1b[0m  16.0Gi       4.5Gi       8.0Gi       0.5Gi       3.0Gi      11.0Gi\n" +
				"\x1b[1mSwap:         \x1b[0m   4.0Gi       0.0Gi       4.0Gi", nil
		}
		// Generic formatter response for other commands
		return "\x1b[1m\x1b[36mFormatted output for command:\x1b[0m \x1b[32m" + 
			strings.TrimPrefix(strings.TrimPrefix(promptLower, "format this terminal output from the command '"), "' to make it more readable:") + 
			"\x1b[0m", nil
	}
	
	// Handle analytics requests
	if strings.Contains(promptLower, "analytics") || strings.Contains(promptLower, "directory structure") || 
	   strings.Contains(promptLower, "file contents") || strings.Contains(promptLower, "question") {
		if strings.Contains(promptLower, "code") || strings.Contains(promptLower, "programming") {
			return "## Code Analysis\n\n" +
				"This directory contains:\n" +
				"- **3 Go files** (main.go, types.go, utils.go)\n" +
				"- **2 JavaScript files** (app.js, utils.js)\n" +
				"- **1 Python file** (script.py)\n\n" +
				"The main programming languages used are:\n" +
				"1. Go (50%)\n" +
				"2. JavaScript (30%)\n" +
				"3. Python (20%)\n\n" +
				"The Go code implements a command-line tool with several modules.", nil
		} else if strings.Contains(promptLower, "document") || strings.Contains(promptLower, "markdown") {
			return "## Documentation Analysis\n\n" +
				"The repository contains:\n" +
				"- README.md: Project overview and setup instructions\n" +
				"- CONTRIBUTING.md: Guidelines for contributors\n" +
				"- docs/: Directory with detailed documentation\n\n" +
				"Key topics covered in documentation:\n" +
				"1. Installation instructions\n" +
				"2. API reference\n", nil
		} else {
			return "## Directory Analysis\n\n" +
				"This directory contains:\n" +
				"- 12 files (total size: 342KB)\n" +
				"- 4 subdirectories\n\n" +
				"File types:\n" +
				"- Source code: 8 files\n" +
				"- Documentation: 3 files\n" +
				"- Configuration: 1 file\n\n" +
				"The project appears to be a command-line tool written primarily in Go.", nil
		}
	}
	
	// Handle direct response requests
	if strings.Contains(promptLower, "file system") || strings.Contains(promptLower, "question about") {
		if strings.Contains(promptLower, "symbolic link") || strings.Contains(promptLower, "symlink") {
			return "A symbolic link (also called a symlink or soft link) is a special type of file that points to another file or directory. " +
				"Unlike a hard link, which points directly to the data on disk, a symbolic link contains a path that identifies another file or directory.\n\n" +
				"In a directory listing (ls -la), symbolic links are indicated with an \"l\" at the beginning of the permissions string, " +
				"and they show which file or directory they point to.", nil
		} else if strings.Contains(promptLower, "chmod") || strings.Contains(promptLower, "permission") {
			return "The chmod command in Linux/Unix changes the permissions of files and directories.\n\n" +
				"File permissions have three basic levels:\n" +
				"- Read (r): Permission to read the file/list directory contents\n" +
				"- Write (w): Permission to modify the file/add or remove files in a directory\n" +
				"- Execute (x): Permission to run the file as a program/access files in a directory", nil
		} else {
			return "The current directory is where you're currently located in the file system. " +
				"You can see its path with the pwd command.\n\n" +
				"To interact with files in the current directory:\n" +
				"- List files: ls -la\n" +
				"- Create a new directory: mkdir new_dir\n" +
				"- Create a new file: touch new_file.txt", nil
		}
	}
	
	return "I don't understand the request.", nil
}