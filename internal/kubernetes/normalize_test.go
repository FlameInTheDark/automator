package kubernetes

import (
	"strings"
	"testing"
)

const sampleKubeconfig = `
apiVersion: v1
kind: Config
current-context: dev
clusters:
- name: local
  cluster:
    server: https://cluster.example
contexts:
- name: dev
  context:
    cluster: local
    user: deployer
    namespace: team-a
users:
- name: deployer
  user:
    token: secret-token
`

func TestNormalizeClusterInputFromKubeconfig(t *testing.T) {
	t.Parallel()

	normalized, err := NormalizeClusterInput(ClusterInput{
		Name:       "Dev",
		SourceType: SourceTypeKubeconfig,
		Kubeconfig: sampleKubeconfig,
	})
	if err != nil {
		t.Fatalf("NormalizeClusterInput returned error: %v", err)
	}

	if normalized.SourceType != SourceTypeKubeconfig {
		t.Fatalf("SourceType = %q, want %q", normalized.SourceType, SourceTypeKubeconfig)
	}
	if normalized.ContextName != "dev" {
		t.Fatalf("ContextName = %q, want dev", normalized.ContextName)
	}
	if normalized.DefaultNamespace != "team-a" {
		t.Fatalf("DefaultNamespace = %q, want team-a", normalized.DefaultNamespace)
	}
	if normalized.Server != "https://cluster.example" {
		t.Fatalf("Server = %q, want https://cluster.example", normalized.Server)
	}
}

func TestNormalizeClusterInputManualRoundTrip(t *testing.T) {
	t.Parallel()

	normalized, err := NormalizeClusterInput(ClusterInput{
		Name:        "Prod",
		SourceType:  SourceTypeManual,
		ContextName: "prod-admin",
		Manual: &ManualAuthConfig{
			Server:                "https://prod.example",
			Token:                 "prod-token",
			CAData:                "CA CERT",
			ClientCertificateData: "CLIENT CERT",
			ClientKeyData:         "CLIENT KEY",
		},
	})
	if err != nil {
		t.Fatalf("NormalizeClusterInput returned error: %v", err)
	}

	if normalized.SourceType != SourceTypeManual {
		t.Fatalf("SourceType = %q, want %q", normalized.SourceType, SourceTypeManual)
	}
	if normalized.ContextName != "prod-admin" {
		t.Fatalf("ContextName = %q, want prod-admin", normalized.ContextName)
	}
	if normalized.DefaultNamespace != "default" {
		t.Fatalf("DefaultNamespace = %q, want default", normalized.DefaultNamespace)
	}
	if !strings.Contains(normalized.Kubeconfig, "prod-admin") {
		t.Fatalf("expected generated kubeconfig to contain context name, got %q", normalized.Kubeconfig)
	}

	manual, namespace, err := RecoverManualConfig(normalized.Kubeconfig, normalized.ContextName)
	if err != nil {
		t.Fatalf("RecoverManualConfig returned error: %v", err)
	}
	if manual.Server != "https://prod.example" {
		t.Fatalf("manual.Server = %q, want https://prod.example", manual.Server)
	}
	if manual.Token != "prod-token" {
		t.Fatalf("manual.Token = %q, want prod-token", manual.Token)
	}
	if namespace != "default" {
		t.Fatalf("namespace = %q, want default", namespace)
	}
}

func TestNormalizeClusterInputRejectsUnknownContextOverride(t *testing.T) {
	t.Parallel()

	_, err := NormalizeClusterInput(ClusterInput{
		Name:        "Dev",
		SourceType:  SourceTypeKubeconfig,
		Kubeconfig:  sampleKubeconfig,
		ContextName: "missing",
	})
	if err == nil {
		t.Fatal("expected unknown context override to fail")
	}
	if !strings.Contains(err.Error(), `context "missing" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
