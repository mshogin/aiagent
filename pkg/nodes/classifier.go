package nodes

import (
	"encoding/json"
	"fmt"
)

// ClassifierNodeInterface defines the operations for a classifier node
type ClassifierNodeInterface interface {
	// Process analyzes the input and determines which node should process it next
	//
	// Parameters:
	//   - state: The current state object that contains all information shared between nodes
	//
	// Returns:
	//   - error: An error if processing fails
	Process(state *State) error
}

// ClassifierNode is responsible for determining which node should process the state next
type ClassifierNode struct {
	llm LLM
}

// NewClassifierNode creates a new instance of ClassifierNode
func NewClassifierNode(llm LLM) *ClassifierNode {
	return &ClassifierNode{
		llm: llm,
	}
}

type classifierResponse struct {
	NextNode    NodeType `json:"next_node"`
	Goal        string   `json:"goal"`
	IsTaskDone  bool     `json:"is_task_done,omitempty"`
	IsGoalMet   bool     `json:"is_goal_met,omitempty"`
	Explanation string   `json:"explanation"`
}

// Process implements the Node interface for ClassifierNode
func (n *ClassifierNode) Process(state *State) (string, error) {
	// If there's a current task, verify if it's completed
	if state.CurrentTask.NodeType != "" {
		completed, err := n.verifyTaskCompletion(state)
		if err != nil {
			return "", fmt.Errorf("failed to verify task completion: %v", err)
		}

		state.CurrentTask.IsCompleted = completed

		if completed {
			// Add completed task to history
			state.TaskHistory = append(state.TaskHistory, state.CurrentTask)

			// Check if global goal is met
			goalMet, err := n.isGlobalGoalMet(state)
			if err != nil {
				return "", fmt.Errorf("failed to check global goal: %v", err)
			}

			if goalMet {
				state.NextNode = NodeTypeTerminal
				state.CurrentTask = TaskStatus{}
				return "", nil
			}
		}
	}

	// Get next node and goal
	nextNode, goal, err := n.classifyRequest(state)
	if err != nil {
		return "", fmt.Errorf("failed to classify request: %v", err)
	}

	// Update state
	state.NextNode = nextNode
	state.CurrentTask = TaskStatus{
		NodeType: nextNode,
		Goal:     goal,
	}

	return goal, nil
}

func (n *ClassifierNode) verifyTaskCompletion(state *State) (bool, error) {
	prompt := fmt.Sprintf(`Verify if the following task was completed successfully:
Task Goal: %s
Node Type: %s
Result: %s

Please analyze if the task goal was achieved based on the result.
Return JSON response with:
{
    "is_task_done": boolean,
    "explanation": "why the task is considered done or not"
}`, state.CurrentTask.Goal, state.CurrentTask.NodeType, state.CurrentTask.Result)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return false, fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		IsTaskDone  bool   `json:"is_task_done"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return false, fmt.Errorf("failed to parse LLM response: %v", err)
	}

	return result.IsTaskDone, nil
}

func (n *ClassifierNode) isGlobalGoalMet(state *State) (bool, error) {
	prompt := fmt.Sprintf(`Based on the completed tasks and current state, determine if the global goal has been met:
Global Goal: %s
Completed Tasks: %v
Current State: `, state.GlobalGoal, state.TaskHistory)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return false, fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		IsGoalMet   bool   `json:"is_goal_met"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return false, fmt.Errorf("failed to parse LLM response: %v", err)
	}

	return result.IsGoalMet, nil
}

func (n *ClassifierNode) classifyRequest(state *State) (NodeType, string, error) {
	prompt := fmt.Sprintf(`Based on the current state and task history, determine the next node to process the request:
Input: %s
Global Goal: %s
Task History: %v
Current State: `, state.Input, state.GlobalGoal, state.TaskHistory)

	response, err := n.llm.Complete(prompt)
	if err != nil {
		return "", "", fmt.Errorf("LLM error: %v", err)
	}

	var result struct {
		NextNode    string `json:"next_node"`
		Goal        string `json:"goal"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "", "", fmt.Errorf("failed to parse LLM response: %v", err)
	}

	return NodeType(result.NextNode), result.Goal, nil
}

func (n *ClassifierNode) Type() NodeType {
	return NodeTypeClassifier
}
