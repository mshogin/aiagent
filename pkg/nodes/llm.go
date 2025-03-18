package nodes

import (
	"fmt"
)

// LLM defines the interface for language model interactions
type LLM interface {
	Complete(prompt string) (string, error)
}

// MockLLMForTesting implements LLM interface for testing
type MockLLMForTesting struct {
	Responses map[string]string
}

func (m *MockLLMForTesting) Complete(prompt string) (string, error) {
	if response, ok := m.Responses[prompt]; ok {
		return response, nil
	}
	// Debug output to help identify mismatched prompts
	fmt.Printf("No response found for prompt:\n%q\n\nAvailable prompts:\n", prompt)
	for p := range m.Responses {
		fmt.Printf("%q\n", p)
	}
	return "", nil
}
