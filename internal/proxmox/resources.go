package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
)

type Container struct {
	CTID   int    `json:"vmid"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	CPU    int    `json:"cpus,omitempty"`
	Memory int    `json:"maxmem,omitempty"`
	Disk   int    `json:"maxdisk,omitempty"`
	Node   string `json:"node,omitempty"`
	Type   string `json:"type,omitempty"`
}

type Storage struct {
	Storage string `json:"storage"`
	Type    string `json:"type"`
	Used    int64  `json:"used"`
	Total   int64  `json:"total"`
	Avail   int64  `json:"avail"`
	Content string `json:"content"`
	Active  int    `json:"active"`
	Enabled int    `json:"enabled"`
	Shared  int    `json:"shared"`
}

type TaskStatus struct {
	UPID      string `json:"upid"`
	Node      string `json:"node"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	User      string `json:"user"`
	Status    string `json:"status"`
	ExitCode  string `json:"exitstatus"`
	StartTime int64  `json:"starttime"`
	EndTime   int64  `json:"endtime"`
}

func (c *Client) ListContainers(ctx context.Context, node string) ([]Container, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/nodes/%s/lxc", node))
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var containers []Container
	if err := json.Unmarshal(data, &containers); err != nil {
		return nil, fmt.Errorf("unmarshal containers: %w", err)
	}

	return containers, nil
}

func (c *Client) StartContainer(ctx context.Context, node string, ctid int) error {
	_, err := c.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/start", node, ctid), nil)
	return err
}

func (c *Client) StopContainer(ctx context.Context, node string, ctid int) error {
	_, err := c.Post(ctx, fmt.Sprintf("/nodes/%s/lxc/%d/status/stop", node, ctid), nil)
	return err
}

func (c *Client) ListStorages(ctx context.Context, node string) ([]Storage, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/nodes/%s/storage", node))
	if err != nil {
		return nil, fmt.Errorf("list storages: %w", err)
	}

	var storages []Storage
	if err := json.Unmarshal(data, &storages); err != nil {
		return nil, fmt.Errorf("unmarshal storages: %w", err)
	}

	return storages, nil
}

func (c *Client) GetTaskStatus(ctx context.Context, node, upid string) (*TaskStatus, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/nodes/%s/tasks/%s/status", node, upid))
	if err != nil {
		return nil, fmt.Errorf("get task status: %w", err)
	}

	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal task status: %w", err)
	}

	return &status, nil
}

func (c *Client) ListClusterResources(ctx context.Context) ([]map[string]interface{}, error) {
	data, err := c.Get(ctx, "/cluster/resources")
	if err != nil {
		return nil, fmt.Errorf("list cluster resources: %w", err)
	}

	var resources []map[string]interface{}
	if err := json.Unmarshal(data, &resources); err != nil {
		return nil, fmt.Errorf("unmarshal cluster resources: %w", err)
	}

	return resources, nil
}
