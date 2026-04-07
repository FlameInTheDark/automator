package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
)

type VM struct {
	VMID   int    `json:"vmid"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	CPU    int    `json:"cpus,omitempty"`
	Memory int    `json:"maxmem,omitempty"`
	Disk   int    `json:"maxdisk,omitempty"`
	Node   string `json:"node,omitempty"`
	Type   string `json:"type,omitempty"`
}

func (c *Client) ListVMs(ctx context.Context, node string) ([]VM, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/nodes/%s/qemu", node))
	if err != nil {
		return nil, fmt.Errorf("list vms: %w", err)
	}

	var vms []VM
	if err := json.Unmarshal(data, &vms); err != nil {
		return nil, fmt.Errorf("unmarshal vms: %w", err)
	}

	return vms, nil
}

func (c *Client) GetVM(ctx context.Context, node string, vmid int) (*VM, error) {
	data, err := c.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/current", node, vmid))
	if err != nil {
		return nil, fmt.Errorf("get vm: %w", err)
	}

	var vm VM
	if err := json.Unmarshal(data, &vm); err != nil {
		return nil, fmt.Errorf("unmarshal vm: %w", err)
	}

	return &vm, nil
}

func (c *Client) StartVM(ctx context.Context, node string, vmid int) error {
	_, err := c.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/start", node, vmid), nil)
	return err
}

func (c *Client) StopVM(ctx context.Context, node string, vmid int) error {
	_, err := c.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", node, vmid), nil)
	return err
}

func (c *Client) ShutdownVM(ctx context.Context, node string, vmid int) error {
	_, err := c.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/shutdown", node, vmid), nil)
	return err
}

func (c *Client) CloneVM(ctx context.Context, node string, vmid int, newName string, newID int) error {
	payload := map[string]interface{}{
		"newid": newID,
		"name":  newName,
		"full":  1,
	}
	_, err := c.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/clone", node, vmid), payload)
	return err
}

func (c *Client) DeleteVM(ctx context.Context, node string, vmid int) error {
	_, err := c.Delete(ctx, fmt.Sprintf("/nodes/%s/qemu/%d", node, vmid))
	return err
}
