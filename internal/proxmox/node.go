package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
)

type Node struct {
	Node        string  `json:"node"`
	Status      string  `json:"status"`
	CPU         float64 `json:"cpu"`
	MemoryUsed  int64   `json:"mem"`
	MemoryTotal int64   `json:"maxmem"`
	DiskUsed    int64   `json:"disk"`
	DiskTotal   int64   `json:"maxdisk"`
	Uptime      int64   `json:"uptime"`
}

func (c *Client) ListNodes(ctx context.Context) ([]Node, error) {
	data, err := c.Get(ctx, "/nodes")
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	var nodes []Node
	if err := json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("unmarshal nodes: %w", err)
	}

	return nodes, nil
}

func (c *Client) GetNodeStatus(ctx context.Context, node string) (*Node, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/nodes/%s/status", node))
	if err != nil {
		return nil, fmt.Errorf("get node status: %w", err)
	}

	var n Node
	if err := json.Unmarshal(data, &n); err != nil {
		return nil, fmt.Errorf("unmarshal node: %w", err)
	}

	return &n, nil
}
