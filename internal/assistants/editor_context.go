package assistants

type ExecutionLogAttachment struct {
	ID        string                       `json:"id"`
	Execution ExecutionLogAttachmentRun    `json:"execution"`
	Nodes     []ExecutionLogAttachmentNode `json:"nodes"`
}

type ExecutionLogAttachmentRun struct {
	ID          string `json:"id"`
	TriggerType string `json:"trigger_type"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	Error       string `json:"error,omitempty"`
}

type ExecutionLogAttachmentNode struct {
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"`
	Status   string `json:"status"`
	Input    any    `json:"input,omitempty"`
	Output   any    `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}
