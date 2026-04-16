package cli

import (
	"testing"

	"github.com/FlameInTheDark/emerald/internal/node"
)

func TestBuildNodeRegistryIncludesSharedCLIAndServerNodes(t *testing.T) {
	t.Parallel()

	registry := buildNodeRegistry(registryDependencies{})
	types := registry.ListTypes()

	for _, required := range []node.NodeType{
		node.TypeLLMAgent,
		node.TypeActionShell,
		node.TypeToolShell,
		node.TypeActionRunPipeline,
		node.TypeToolRunPipeline,
		node.TypeToolCreatePipeline,
		node.TypeToolUpdatePipeline,
		node.TypeToolDeletePipeline,
		node.TypeActionListNodes,
		node.TypeToolListNodes,
		node.TypeActionKubernetesListResources,
		node.TypeToolKubernetesListResources,
	} {
		if !containsNodeType(types, required) {
			t.Fatalf("registry missing required node type %s", required)
		}
	}
}

func containsNodeType(types []node.NodeType, want node.NodeType) bool {
	for _, nodeType := range types {
		if nodeType == want {
			return true
		}
	}
	return false
}
