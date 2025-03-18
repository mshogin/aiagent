package nodes

import (
	"testing"
)

func TestState_TaskManagement(t *testing.T) {
	tests := []struct {
		name            string
		state           *State
		task            TaskStatus
		expectedHistory []TaskStatus
	}{
		{
			name: "add first task",
			state: &State{
				TaskHistory: make([]TaskStatus, 0),
			},
			task: TaskStatus{
				NodeType:    NodeTypeBash,
				Goal:        "list files",
				IsCompleted: true,
				Result:      "file1.txt\nfile2.txt",
			},
			expectedHistory: []TaskStatus{
				{
					NodeType:    NodeTypeBash,
					Goal:        "list files",
					IsCompleted: true,
					Result:      "file1.txt\nfile2.txt",
				},
			},
		},
		{
			name: "add multiple tasks",
			state: &State{
				TaskHistory: []TaskStatus{
					{
						NodeType:    NodeTypeBash,
						Goal:        "list files",
						IsCompleted: true,
						Result:      "file1.txt\nfile2.txt",
					},
				},
			},
			task: TaskStatus{
				NodeType:    NodeTypeFormatter,
				Goal:        "format output",
				IsCompleted: true,
				Result:      "Formatted output",
			},
			expectedHistory: []TaskStatus{
				{
					NodeType:    NodeTypeBash,
					Goal:        "list files",
					IsCompleted: true,
					Result:      "file1.txt\nfile2.txt",
				},
				{
					NodeType:    NodeTypeFormatter,
					Goal:        "format output",
					IsCompleted: true,
					Result:      "Formatted output",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add task to history
			tt.state.TaskHistory = append(tt.state.TaskHistory, tt.task)

			// Check history length
			if len(tt.state.TaskHistory) != len(tt.expectedHistory) {
				t.Errorf("TaskHistory length = %v, expected %v", len(tt.state.TaskHistory), len(tt.expectedHistory))
			}

			// Check each task in history
			for i, task := range tt.state.TaskHistory {
				if task.NodeType != tt.expectedHistory[i].NodeType {
					t.Errorf("Task[%d].NodeType = %v, expected %v", i, task.NodeType, tt.expectedHistory[i].NodeType)
				}
				if task.Goal != tt.expectedHistory[i].Goal {
					t.Errorf("Task[%d].Goal = %v, expected %v", i, task.Goal, tt.expectedHistory[i].Goal)
				}
				if task.IsCompleted != tt.expectedHistory[i].IsCompleted {
					t.Errorf("Task[%d].IsCompleted = %v, expected %v", i, task.IsCompleted, tt.expectedHistory[i].IsCompleted)
				}
				if task.Result != tt.expectedHistory[i].Result {
					t.Errorf("Task[%d].Result = %v, expected %v", i, task.Result, tt.expectedHistory[i].Result)
				}
			}
		})
	}
}

func TestState_FileContentManagement(t *testing.T) {
	tests := []struct {
		name          string
		state         *State
		fileContent   FileContent
		expectedFiles []FileContent
	}{
		{
			name: "add first file",
			state: &State{
				DirectoryContents: make([]FileContent, 0),
			},
			fileContent: FileContent{
				Path:    "test.txt",
				Content: "test content",
				Size:    100,
				IsDir:   false,
			},
			expectedFiles: []FileContent{
				{
					Path:    "test.txt",
					Content: "test content",
					Size:    100,
					IsDir:   false,
				},
			},
		},
		{
			name: "add multiple files",
			state: &State{
				DirectoryContents: []FileContent{
					{
						Path:    "test.txt",
						Content: "test content",
						Size:    100,
						IsDir:   false,
					},
				},
			},
			fileContent: FileContent{
				Path:    "dir1",
				Content: "",
				Size:    0,
				IsDir:   true,
			},
			expectedFiles: []FileContent{
				{
					Path:    "test.txt",
					Content: "test content",
					Size:    100,
					IsDir:   false,
				},
				{
					Path:    "dir1",
					Content: "",
					Size:    0,
					IsDir:   true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add file to directory contents
			tt.state.DirectoryContents = append(tt.state.DirectoryContents, tt.fileContent)

			// Check directory contents length
			if len(tt.state.DirectoryContents) != len(tt.expectedFiles) {
				t.Errorf("DirectoryContents length = %v, expected %v", len(tt.state.DirectoryContents), len(tt.expectedFiles))
			}

			// Check each file in directory contents
			for i, file := range tt.state.DirectoryContents {
				if file.Path != tt.expectedFiles[i].Path {
					t.Errorf("File[%d].Path = %v, expected %v", i, file.Path, tt.expectedFiles[i].Path)
				}
				if file.Content != tt.expectedFiles[i].Content {
					t.Errorf("File[%d].Content = %v, expected %v", i, file.Content, tt.expectedFiles[i].Content)
				}
				if file.Size != tt.expectedFiles[i].Size {
					t.Errorf("File[%d].Size = %v, expected %v", i, file.Size, tt.expectedFiles[i].Size)
				}
				if file.IsDir != tt.expectedFiles[i].IsDir {
					t.Errorf("File[%d].IsDir = %v, expected %v", i, file.IsDir, tt.expectedFiles[i].IsDir)
				}
			}
		})
	}
}
