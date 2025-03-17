# AI Agent CLI Tool

A command-line interface tool that uses AI to assist with various tasks including code analysis, command generation, and more.

## Disclaimer

**This codebase was completely developed by Claude Code (Anthropic's Claude AI assistant). The author is not responsible for any issues that may arise from its use.**

We are planning to continue developing this codebase using an agent-only approach, with minimal human intervention.

## Features

* Command generation and execution with safety validation
* Code analysis and insights
* Content collection and summarization
* Directory structure analysis
* Rich terminal output formatting

## Installation

```bash
# Clone the repository
git clone https://github.com/your-username/aiagent.git
cd aiagent

# Build the application
go build -o aiagent cmd/aiagent/main.go
```

## Usage

```bash
# Basic usage
./aiagent "your request here"

# Use mock LLM (no API key needed)
./aiagent --mock "your request here"

# Enable verbose mode
./aiagent -v "your request here"

# Force approve commands (use with caution)
./aiagent -y "your request here"
```

## Examples

```bash
# Generate and execute a command
./aiagent "show me the disk usage in a readable format"

# Analyze code
./aiagent "collect all information about the formatter component"

# Get a direct answer
./aiagent "explain what chmod does"
```

## Configuration

By default, the application uses the OpenAI API. You need to set the `OPENAI_API_KEY` environment variable:

```bash
export OPENAI_API_KEY="your-api-key"
```

## License

This project is open source and available under the [MIT License](LICENSE).