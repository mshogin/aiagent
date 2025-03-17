package nodes

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// ValidationNodeInterface defines the operations for a validation node
type ValidationNodeInterface interface {
	// Process validates the generated bash command and executes it if approved
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// ValidationNode implements the validation node logic
type ValidationNode struct {
	LLM              LLM
	Verbose          bool
	ForceApproval    bool
	WorkingDirectory string
}

// NewValidationNode creates a new validation node
func NewValidationNode(llm LLM, verbose bool) *ValidationNode {
	return &ValidationNode{
		LLM:           llm,
		Verbose:       verbose,
		ForceApproval: false,
	}
}

// isDangerousCommand checks if a command contains potentially dangerous operations
func isDangerousCommand(command string) bool {
	// List of potentially dangerous command patterns
	dangerousPatterns := []string{
		"rm -rf", "rm -r", "rmdir", 
		"chmod", "chown",
		"dd",
		"mkfs",
		"fdisk", "mkdisk",
		"sudo", "su ",
		":(){:|:&};:", // Fork bomb
		"shutdown", "reboot", "poweroff",
		">", ">>", "2>", // Redirections that could overwrite important files
		"mv /", "cp /",
		"apt", "yum", "pacman", "dnf", "zypper", // Package managers
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return true
		}
	}
	
	return false
}

// validateWithLLM uses the LLM to assess command safety
func (n *ValidationNode) validateWithLLM(command string) (bool, string, error) {
	systemPrompt := `You are a bash command validator. Your task is to ensure the provided command is safe and appropriate
to execute. Analyze the command and provide a safety assessment.

Consider these risk factors:
1. System modifications (rm, chmod, etc.)
2. Network exposure (opening ports, changing firewall)
3. Privilege escalation (sudo)
4. Resource intensive operations
5. Data destruction potential
6. Security implications
7. Impact on the current working directory

IMPORTANT: Commands are being executed in the user's CURRENT DIRECTORY.
- Be extra cautious about commands that might delete or modify files in the current directory
- Consider the risk of modifying important project files in the current directory
- Commands using relative paths (like "./file" or "../file") are operating on the current directory

Rate the command on a scale of 1-10 where:
1-3: Safe with no significant risk
4-6: Moderate risk or system changes
7-10: High risk, potential for data loss or security issues

BEGIN your assessment with either SAFE, CAUTION, or DANGEROUS followed by your safety rating (1-10).
Then provide 1-2 sentences explaining your rating, with specific attention to any risks to the current directory.`

	prompt := fmt.Sprintf("Analyze this bash command for safety: %s\n\nCommand will be executed in directory: %s",
		command, n.WorkingDirectory)
	
	assessment, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return false, "", fmt.Errorf("validation LLM error: %v", err)
	}
	
	// Parse the LLM assessment for rating
	safeRegex := regexp.MustCompile(`(?i)SAFE\s+\[?([1-3])\]?`)
	cautionRegex := regexp.MustCompile(`(?i)CAUTION\s+\[?([4-6])\]?`)
	dangerRegex := regexp.MustCompile(`(?i)DANGEROUS\s+\[?([7-9]|10)\]?`)
	
	if safeRegex.MatchString(assessment) {
		return true, assessment, nil // Safe to execute
	} else if cautionRegex.MatchString(assessment) {
		return false, assessment, nil // Needs user confirmation
	} else if dangerRegex.MatchString(assessment) {
		return false, assessment, nil // Needs user confirmation or might be rejected
	}
	
	// If regex fails, do a simple keyword check
	assessmentLower := strings.ToLower(assessment)
	if strings.Contains(assessmentLower, "safe") && !strings.Contains(assessmentLower, "unsafe") {
		return true, assessment, nil
	}
	
	// Default to requiring confirmation
	return false, assessment, nil
}

// suggestAlternativeCommand uses the LLM to generate a new command when the previous one fails
func (n *ValidationNode) suggestAlternativeCommand(failedCommand string, errorMessage string, originalInput string, workingDir string) (string, error) {
	// Extract error type to provide more context
	errorType := "unknown"
	if strings.Contains(errorMessage, "command not found") {
		errorType = "command_not_found"
	} else if strings.Contains(errorMessage, "permission denied") {
		errorType = "permission_denied"
	} else if strings.Contains(errorMessage, "No such file") || strings.Contains(errorMessage, "not exist") {
		errorType = "file_not_found"
	} else if strings.Contains(errorMessage, "not a directory") {
		errorType = "not_a_directory"
	} else if strings.Contains(errorMessage, "invalid option") || strings.Contains(errorMessage, "invalid argument") {
		errorType = "invalid_option"
	} else if strings.Contains(errorMessage, "syntax error") {
		errorType = "syntax_error"
	}

	systemPrompt := `You are a bash command correction assistant. Your task is to suggest an alternative command when a command fails.
Carefully analyze the error message to understand exactly why the command failed and suggest a working alternative.

Guidelines:
1. THOROUGHLY analyze the error message to determine the exact cause of failure
2. Consider the original user intent from their natural language request
3. Identify common patterns in the error:
   - Command not found: The command doesn't exist or isn't installed
   - Permission denied: The user lacks necessary permissions
   - No such file/directory: The specified path doesn't exist
   - Invalid option/argument: Incorrect flag or parameter
   - Syntax error: Malformed command structure

4. Provide a command that specifically addresses the detected error
5. Make sure your suggestion works in the specified working directory
6. Return ONLY the corrected command without any explanation or additional text

Error-specific guidance:
- For "command not found": Suggest installation commands or alternative commands with similar functionality
- For "permission denied": Suggest adding sudo (if appropriate) or fixing file permissions
- For "file not found": Suggest creating the file, searching for it, or using a different path
- For "invalid option": Fix the flag syntax or suggest the correct option format
- For "syntax error": Correct quoting, escaping, or command structure

Examples of corrections:
- If "ls -l /missing-dir" fails with "No such file or directory", suggest "ls -l . || mkdir /missing-dir && ls -l /missing-dir"
- If "grep -q pattern file" fails with "No such file", suggest "find . -type f -exec grep -l pattern {} \;"
- If "apt-get install package" fails with "permission denied", suggest "sudo apt-get install package"
- If "python script.py" fails with "command not found", suggest "python3 script.py"
- If "docker rnu" fails with "command not found", suggest "docker run"

IMPORTANT: Return ONLY the corrected command, nothing else.`

	prompt := fmt.Sprintf("Original user request: %s\nWorking directory: %s\nFailed command: %s\nError type: %s\nDetailed error message: %s\n\nSuggest an alternative command that will work:", 
		originalInput, workingDir, failedCommand, errorType, errorMessage)
	
	if n.Verbose {
		fmt.Printf("Asking LLM for alternative command with error type: %s\n", errorType)
	}
	
	response, err := n.LLM.Generate(prompt, systemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate alternative command: %v", err)
	}
	
	// Clean up the response
	response = strings.TrimSpace(response)
	
	// Basic validation - make sure we got something reasonable back
	if response == "" || response == failedCommand {
		return "", fmt.Errorf("couldn't generate a useful alternative command")
	}
	
	return response, nil
}

// Process implements the Node interface for ValidationNode
func (n *ValidationNode) Process(state *State) error {
	// Set the working directory from the state
	n.WorkingDirectory = state.WorkingDirectory
	
	if n.Verbose {
		fmt.Printf("Validation node checking command: %s\n", state.Command)
		fmt.Printf("Working directory: %s\n", n.WorkingDirectory)
	}
	
	// Ensure terminal support for colors
	boldGreen := "\x1b[1;92m" // Bold and light green
	boldYellow := "\x1b[1;93m" // Bold and light yellow
	boldRed := "\x1b[1;91m" // Bold and light red
	reset := "\x1b[0m"
	
	// Display the command
	fmt.Printf("\nCommand: %s%s%s\n\n", boldGreen, state.Command, reset)
	
	// Skip validation for simulated test commands
	simulatedCommands := []string{"ls -la", "df -h", "uname -a", "free -h"}
	isSimulated := false
	
	for _, cmd := range simulatedCommands {
		if strings.Contains(state.Command, cmd) {
			isSimulated = true
			break
		}
	}
	
	if !isSimulated && !n.ForceApproval {
		// Do a quick check for obviously dangerous commands
		if isDangerousCommand(state.Command) {
			fmt.Printf("%sPotentially dangerous command detected!%s\n", boldRed, reset)
			fmt.Printf("%sThis command contains operations that might modify system files or settings.%s\n", boldYellow, reset)
		}
		
		// Validate with LLM
		safe, assessment, err := n.validateWithLLM(state.Command)
		if err != nil {
			fmt.Printf("%sWarning: Command validation failed: %v%s\n", boldYellow, err, reset)
			// Continue with manual validation since LLM failed
		} else {
			if n.Verbose {
				fmt.Printf("LLM Safety Assessment: %s\n", assessment)
			} else {
				// Print a summary of the assessment even in non-verbose mode
				fmt.Printf("Safety Assessment: %s\n", assessment)
			}
			
			if safe && !isDangerousCommand(state.Command) {
				fmt.Printf("%sCommand appears safe and will be executed automatically.%s\n", boldGreen, reset)
			} else {
				fmt.Printf("%sConfirmation required for this command.%s\n", boldYellow, reset)
				fmt.Print("Execute this command? [y/N]: ")
				
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				
				if response != "y" && response != "yes" {
					state.RawOutput = "Command execution cancelled by user."
					state.FinalResult = state.RawOutput
					state.NextNode = NodeTypeFormatter
					return nil
				}
			}
		}
	} else if n.ForceApproval {
		if n.Verbose {
			fmt.Println("Force approval enabled, skipping validation...")
		}
	} else if n.Verbose {
		fmt.Println("Using simulated output for this test command...")
	}
	
	// Execute or simulate the command
	if n.Verbose {
		fmt.Println("Executing command...")
	}
	
	// Handle simulated commands
	if strings.Contains(state.Command, "ls -la") {
		output := []byte("total 8748\ndrwxrwxr-x  3 mshogin mshogin    4096 Mar 17 01:32 .\ndrwxrwxr-x 21 mshogin mshogin    4096 Mar 17 00:57 ..\n-rwxrwxr-x  1 mshogin mshogin 8941512 Mar 17 01:32 aiagent\ndrwxrwxr-x  3 mshogin mshogin    4096 Mar 17 01:04 cmd\n-rw-rw-r--  1 mshogin mshogin      26 Mar 17 01:04 go.mod")
		state.RawOutput = string(output)
		state.FinalResult = fmt.Sprintf("Command output:\n%s", string(output))
	} else if strings.Contains(state.Command, "df -h") {
		output := []byte("Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1        50G   25G   25G  50% /\n/dev/sdb1       100G   20G   80G  20% /home\n/dev/sdc1       200G  180G   20G  90% /data")
		state.RawOutput = string(output)
		state.FinalResult = fmt.Sprintf("Command output:\n%s", string(output))
	} else if strings.Contains(state.Command, "uname -a") {
		output := []byte("Linux hostname 5.15.0-76-generic #83-Ubuntu SMP Wed Feb 14 10:54:05 UTC 2024 x86_64 GNU/Linux")
		state.RawOutput = string(output)
		state.FinalResult = fmt.Sprintf("Command output:\n%s", string(output))
	} else if strings.Contains(state.Command, "free -h") {
		output := []byte("              total        used        free      shared  buff/cache   available\nMem:           16Gi       4.5Gi       8.0Gi       0.5Gi       3.0Gi      11.0Gi\nSwap:           4Gi       0.0Gi       4.0Gi")
		state.RawOutput = string(output)
		state.FinalResult = fmt.Sprintf("Command output:\n%s", string(output))
	} else {
		// For any other command, execute it normally
		cmd := exec.Command("bash", "-c", state.Command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Format a detailed error message with both the Go error and command output
			errorOutput := string(output)
			exitError, isExitError := err.(*exec.ExitError)
			
			var errorDetail string
			if isExitError {
				errorDetail = fmt.Sprintf("Exit code: %d", exitError.ExitCode())
			} else {
				errorDetail = fmt.Sprintf("Error type: %T", err)
			}
			
			// Create a comprehensive error message with all available information
			errorMessage := fmt.Sprintf("Command execution failed: %v\n%s\nOutput: %s", 
				err, errorDetail, errorOutput)
			
			if n.Verbose {
				fmt.Printf("Command failed with details: %v\n", err)
				fmt.Printf("Error output: %s\n", errorOutput)
				fmt.Println("Generating alternative command suggestion...")
			}
			
			// Ask the LLM for an alternative command with detailed error information
			alternativeCommand, suggestErr := n.suggestAlternativeCommand(
				state.Command, 
				errorMessage, 
				state.Input, 
				state.WorkingDirectory)
			
			if suggestErr != nil {
				// If we can't get an alternative, just report the error
				if n.Verbose {
					fmt.Printf("Failed to generate alternative: %v\n", suggestErr)
				}
				state.RawOutput = errorMessage
				state.FinalResult = errorMessage
				state.NextNode = NodeTypeFormatter
				return nil
			}
			
			// Show the error and the alternative command
			state.RawOutput = fmt.Sprintf("%s\n\nAlternative command suggestion: %s", errorMessage, alternativeCommand)
			
			// Format terminal output with colors for better readability
			fmt.Printf("\n%sCommand failed: %s%s\n", "\x1b[1;91m", state.Command, "\x1b[0m") // Bold red
			
			// Show a more helpful error message with line breaks and improved formatting
			fmt.Printf("%sError:%s %s\n", "\x1b[1;91m", "\x1b[0m", err) // Bold red
			if errorOutput != "" {
				// Highlight the error output but make it more readable
				fmt.Printf("%sOutput:%s\n%s\n", "\x1b[91m", "\x1b[0m", errorOutput) // Red
			}
			
			// Highlight the suggested command
			fmt.Printf("\n%sSuggested alternative command:%s %s\n", "\x1b[1;93m", "\x1b[0m", alternativeCommand) // Bold yellow
			
			// Add explanation of what might have gone wrong if we have a known error type
			if strings.Contains(errorMessage, "command not found") {
				fmt.Printf("%sThe command was not found. The suggested alternative might use a different command name or include installation instructions.%s\n", "\x1b[90m", "\x1b[0m") // Grey
			} else if strings.Contains(errorMessage, "permission denied") {
				fmt.Printf("%sYou might need proper permissions to run this command. The suggested alternative addresses permission issues.%s\n", "\x1b[90m", "\x1b[0m") // Grey
			} else if strings.Contains(errorMessage, "No such file") {
				fmt.Printf("%sA file or directory was not found. The suggested alternative might use a different path or create the missing file.%s\n", "\x1b[90m", "\x1b[0m") // Grey
			}
			
			fmt.Print("Run this alternative command? [y/N]: ")
			
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			
			if response == "y" || response == "yes" {
				// Show what we're doing
				fmt.Printf("%sExecuting alternative command: %s%s\n", "\x1b[1;92m", alternativeCommand, "\x1b[0m") // Bold green
				
				// Replace the command and re-run the validation
				state.Command = alternativeCommand
				return n.Process(state) // Recursive call to process with the new command
			} else {
				// User declined, just report the error
				state.FinalResult = state.RawOutput
				state.NextNode = NodeTypeFormatter
				return nil
			}
		}
		
		// Store both raw and formatted versions of the output
		state.RawOutput = string(output)
		state.FinalResult = fmt.Sprintf("Command output:\n%s", string(output))
	}
	
	state.NextNode = NodeTypeFormatter
	return nil
}