package nodes

type TerminalNode struct{}

func NewTerminalNode() *TerminalNode {
	return &TerminalNode{}
}

func (n *TerminalNode) Process(state *State) (string, error) {
	// Terminal node just returns the final result
	return state.CurrentTask.Result, nil
}

func (n *TerminalNode) Type() NodeType {
	return NodeTypeTerminal
}
