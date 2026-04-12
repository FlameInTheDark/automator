package pluginapi

import "fmt"

func ErrUnknownNode(nodeID string) error {
	return fmt.Errorf("unknown plugin node %q", nodeID)
}

func ErrTriggerRuntimeUnsupported() error {
	return fmt.Errorf("plugin does not expose a trigger runtime")
}
