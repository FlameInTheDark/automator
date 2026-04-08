// Package kubernetes contains normalized Kubernetes integration helpers used by
// settings APIs, standalone chat tools, and pipeline nodes.
package kubernetes

// ClusterInput describes the untrusted settings payload received from the UI.
type ClusterInput struct {
	Name             string
	SourceType       string
	Kubeconfig       string
	ContextName      string
	DefaultNamespace string
	Manual           *ManualAuthConfig
}

// ManualAuthConfig describes a manually entered Kubernetes connection.
type ManualAuthConfig struct {
	Server                string `json:"server"`
	Token                 string `json:"token"`
	Username              string `json:"username"`
	Password              string `json:"password"`
	CAData                string `json:"ca_data"`
	ClientCertificateData string `json:"client_certificate_data"`
	ClientKeyData         string `json:"client_key_data"`
	InsecureSkipTLSVerify bool   `json:"insecure_skip_tls_verify"`
}

// NormalizedCluster is the stored representation of a Kubernetes cluster.
type NormalizedCluster struct {
	SourceType       string
	Kubeconfig       string
	ContextName      string
	DefaultNamespace string
	Server           string
}

// TestConnectionResult is returned by the draft test connection endpoint.
type TestConnectionResult struct {
	Contexts         []string `json:"contexts"`
	EffectiveContext string   `json:"effective_context"`
	DefaultNamespace string   `json:"default_namespace"`
	Server           string   `json:"server"`
	ServerVersion    string   `json:"server_version"`
}

// APIResourceInfo describes an API resource returned from discovery.
type APIResourceInfo struct {
	GroupVersion string   `json:"groupVersion"`
	Name         string   `json:"name"`
	SingularName string   `json:"singularName"`
	Kind         string   `json:"kind"`
	Namespaced   bool     `json:"namespaced"`
	ShortNames   []string `json:"shortNames,omitempty"`
	Verbs        []string `json:"verbs,omitempty"`
}

// ResourceReference identifies a Kubernetes resource.
type ResourceReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind,omitempty"`
	Resource   string `json:"resource,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name,omitempty"`
}

// ListOptions controls generic resource listing.
type ListOptions struct {
	Namespace     string
	APIVersion    string
	Kind          string
	Resource      string
	LabelSelector string
	FieldSelector string
	AllNamespaces bool
	Limit         int64
}

// GetOptions controls generic resource lookup.
type GetOptions struct {
	Namespace  string
	APIVersion string
	Kind       string
	Resource   string
	Name       string
}

// ApplyOptions controls server-side apply manifest operations.
type ApplyOptions struct {
	Namespace    string
	Manifest     string
	FieldManager string
	Force        bool
}

// PatchOptions controls generic patch operations.
type PatchOptions struct {
	Namespace  string
	APIVersion string
	Kind       string
	Resource   string
	Name       string
	Patch      string
	PatchType  string
}

// DeleteOptions controls generic delete operations.
type DeleteOptions struct {
	Namespace         string
	APIVersion        string
	Kind              string
	Resource          string
	Name              string
	LabelSelector     string
	FieldSelector     string
	PropagationPolicy string
}

// ScaleOptions controls scale operations.
type ScaleOptions struct {
	Namespace  string
	APIVersion string
	Kind       string
	Resource   string
	Name       string
	Replicas   int64
}

// RolloutRestartOptions controls rollout restarts.
type RolloutRestartOptions struct {
	Namespace  string
	APIVersion string
	Kind       string
	Resource   string
	Name       string
}

// RolloutStatusOptions controls rollout status polling.
type RolloutStatusOptions struct {
	Namespace      string
	APIVersion     string
	Kind           string
	Resource       string
	Name           string
	TimeoutSeconds int
}

// PodLogOptions controls pod log collection.
type PodLogOptions struct {
	Namespace    string
	Name         string
	Container    string
	TailLines    int64
	SinceSeconds int64
	Timestamps   bool
	Previous     bool
}

// PodExecOptions controls non-interactive pod exec calls.
type PodExecOptions struct {
	Namespace string
	Name      string
	Container string
	Command   []string
}

// EventOptions controls event listing.
type EventOptions struct {
	Namespace          string
	Limit              int64
	FieldSelector      string
	InvolvedObjectName string
	InvolvedObjectKind string
	InvolvedObjectUID  string
}

// RolloutStatusResult describes rollout completion state.
type RolloutStatusResult struct {
	Ready         bool   `json:"ready"`
	Message       string `json:"message"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Replicas      int64  `json:"replicas"`
	Updated       int64  `json:"updated"`
	ReadyReplicas int64  `json:"readyReplicas"`
}

// PodExecResult captures non-interactive pod exec output.
type PodExecResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}
