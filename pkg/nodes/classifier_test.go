package nodes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifierNode_Process(t *testing.T) {
	mockLLM := &MockLLMForTesting{
		Responses: map[string]string{
			"Verify if the following task was completed successfully:\nTask Goal: analyze code in this directory\nNode Type: code_analyzer\nResult: \n\nPlease analyze if the task goal was achieved based on the result.\nReturn JSON response with:\n{\n    \"is_task_done\": boolean,\n    \"explanation\": \"why the task is considered done or not\"\n}": `{"is_task_done": true, "explanation": "Task completed successfully"}`,
			"Based on the completed tasks and current state, determine if the global goal has been met:\nGlobal Goal: analyze code\nCompleted Tasks: [{NodeType:code_analyzer Goal:analyze code in this directory IsCompleted:false Result:} {NodeType:code_analyzer Goal:analyze code in this directory IsCompleted:true Result:}]\nCurrent State: ":         `{"is_goal_met": true, "explanation": "Global goal has been met"}`,
			"Based on the current state and task history, determine the next node to process the request:\nInput: analyze code in this directory\nGlobal Goal: analyze code\nTask History: [{NodeType:code_analyzer Goal:analyze code in this directory IsCompleted:true Result:}]\nCurrent State: ":                                                          `{"next_node": "terminal", "goal": "", "explanation": "All tasks completed"}`,
		},
	}

	node := NewClassifierNode(mockLLM)
	state := &State{
		Input:      "analyze code in this directory",
		GlobalGoal: "analyze code",
		TaskHistory: []TaskStatus{
			{
				NodeType: NodeTypeCodeAnalyzer,
				Goal:     "analyze code in this directory",
			},
		},
		CurrentTask: TaskStatus{
			NodeType: NodeTypeCodeAnalyzer,
			Goal:     "analyze code in this directory",
		},
	}

	result, err := node.Process(state)
	assert.NoError(t, err)
	assert.Equal(t, "", result)
	assert.Equal(t, NodeTypeTerminal, state.NextNode)
	assert.Equal(t, "", state.CurrentTask.Goal)

	// Test verify task completion
	mockLLM = &MockLLMForTesting{
		Responses: map[string]string{
			"Verify if the following task was completed successfully:\nTask Goal: list files in current directory\nNode Type: code_analyzer\nResult: \n\nPlease analyze if the task goal was achieved based on the result.\nReturn JSON response with:\n{\n    \"is_task_done\": boolean,\n    \"explanation\": \"why the task is considered done or not\"\n}": `{"is_task_done": false, "explanation": "Task not completed yet"}`,
			"Based on the current state and task history, determine the next node to process the request:\nInput: list files\nGlobal Goal: list all files\nTask History: [{NodeType:code_analyzer Goal:list files in current directory IsCompleted:false Result:}]\nCurrent State: ":                                                                           `{"next_node": "code_analyzer", "goal": "retry listing files", "explanation": "Retrying task"}`,
		},
	}

	node = NewClassifierNode(mockLLM)
	state = &State{
		Input:      "list files",
		GlobalGoal: "list all files",
		TaskHistory: []TaskStatus{
			{
				NodeType: NodeTypeCodeAnalyzer,
				Goal:     "list files in current directory",
			},
		},
		CurrentTask: TaskStatus{
			NodeType: NodeTypeCodeAnalyzer,
			Goal:     "list files in current directory",
		},
	}

	result, err = node.Process(state)
	assert.NoError(t, err)
	assert.Equal(t, "retry listing files", result)
	assert.Equal(t, NodeTypeCodeAnalyzer, state.NextNode)
	assert.Equal(t, "retry listing files", state.CurrentTask.Goal)

	// Test code analyzer case
	mockLLM = &MockLLMForTesting{
		Responses: map[string]string{
			"Verify if the following task was completed successfully:\nTask Goal: list files in current directory\nNode Type: code_analyzer\nResult: \n\nPlease analyze if the task goal was achieved based on the result.\nReturn JSON response with:\n{\n    \"is_task_done\": boolean,\n    \"explanation\": \"why the task is considered done or not\"\n}": `{"is_task_done": false, "explanation": "Task not completed yet"}`,
			"Based on the current state and task history, determine the next node to process the request:\nInput: analyze code\nGlobal Goal: analyze code\nTask History: []\nCurrent State: ":                                                                                                                                                                  `{"next_node": "code_analyzer", "goal": "retry with sudo", "explanation": "Need to analyze code"}`,
		},
	}

	node = NewClassifierNode(mockLLM)
	state = &State{
		Input:       "analyze code",
		GlobalGoal:  "analyze code",
		TaskHistory: []TaskStatus{},
		CurrentTask: TaskStatus{
			NodeType: NodeTypeCodeAnalyzer,
			Goal:     "list files in current directory",
		},
	}

	result, err = node.Process(state)
	assert.NoError(t, err)
	assert.Equal(t, "retry with sudo", result)
	assert.Equal(t, NodeTypeCodeAnalyzer, state.NextNode)
	assert.Equal(t, "retry with sudo", state.CurrentTask.Goal)
}
