package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/FlameInTheDark/emerald/internal/db/models"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v3"
)

type cliConfigResourceType string

const (
	cliConfigResourceProxmoxCluster    cliConfigResourceType = "proxmox_cluster"
	cliConfigResourceKubernetesCluster cliConfigResourceType = "kubernetes_cluster"
	cliConfigResourceChannel           cliConfigResourceType = "channel"
	cliConfigResourceLLMProvider       cliConfigResourceType = "llm_provider"
)

type cliConfigReference struct {
	ID   string
	Name string
}

type cliConfigListGroup struct {
	ResourceType cliConfigResourceType `json:"resourceType"`
	Items        []map[string]any      `json:"items"`
}

type cliConfigClusterStore interface {
	List(ctx context.Context) ([]models.Cluster, error)
	GetByID(ctx context.Context, id string) (*models.Cluster, error)
	Update(ctx context.Context, cluster *models.Cluster) error
}

type cliConfigKubernetesClusterStore interface {
	List(ctx context.Context) ([]models.KubernetesCluster, error)
	GetByID(ctx context.Context, id string) (*models.KubernetesCluster, error)
	Update(ctx context.Context, cluster *models.KubernetesCluster) error
}

type cliConfigChannelStore interface {
	List(ctx context.Context) ([]models.Channel, error)
	GetByID(ctx context.Context, id string) (*models.Channel, error)
	Update(ctx context.Context, channel *models.Channel) error
}

type cliConfigLLMProviderStore interface {
	List(ctx context.Context) ([]models.LLMProvider, error)
	GetByID(ctx context.Context, id string) (*models.LLMProvider, error)
	Update(ctx context.Context, provider *models.LLMProvider) error
}

func ListConfigs(ctx context.Context, cmd *cli.Command) error {
	return runConfigListCommand(ctx, cmd.String("resource"), cmd.Bool("json"), os.Stdout, newCLIRuntime)
}

func GetConfig(ctx context.Context, cmd *cli.Command) error {
	return runConfigGetCommand(
		ctx,
		cmd.String("resource"),
		cmd.String("id"),
		cmd.String("name"),
		cmd.Bool("show-secrets"),
		os.Stdout,
		newCLIRuntime,
	)
}

func UpdateConfig(ctx context.Context, cmd *cli.Command) error {
	return runConfigUpdateCommand(
		ctx,
		cmd.String("resource"),
		cmd.String("id"),
		cmd.String("name"),
		cmd.String("patch"),
		cmd.String("patch-file"),
		cmd.Bool("show-secrets"),
		os.Stdout,
		newCLIRuntime,
	)
}

func runConfigListCommand(
	ctx context.Context,
	resource string,
	jsonOutput bool,
	stdout io.Writer,
	runtimeFactory cliRuntimeFactory,
) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if runtimeFactory == nil {
		runtimeFactory = newCLIRuntime
	}

	runtime, err := runtimeFactory(ctx, cliRuntimeOptions{migrate: true})
	if err != nil {
		return err
	}
	defer func() {
		_ = runtime.Close()
	}()

	groups, err := loadConfigGroups(ctx, runtime, resource)
	if err != nil {
		return err
	}

	if jsonOutput {
		if strings.TrimSpace(resource) != "" {
			return writePrettyJSON(stdout, map[string]any{
				"resourceType": string(groups[0].ResourceType),
				"items":        groups[0].Items,
			})
		}

		payload := make(map[string]any, len(groups))
		for _, group := range groups {
			payload[string(group.ResourceType)] = group.Items
		}
		return writePrettyJSON(stdout, payload)
	}

	renderConfigListGroups(stdout, groups)
	return nil
}

func runConfigGetCommand(
	ctx context.Context,
	resource string,
	id string,
	name string,
	showSecrets bool,
	stdout io.Writer,
	runtimeFactory cliRuntimeFactory,
) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if runtimeFactory == nil {
		runtimeFactory = newCLIRuntime
	}

	resourceType, err := normalizeCLIConfigResourceType(resource)
	if err != nil {
		return err
	}

	ref, err := newCLIConfigReference(id, name)
	if err != nil {
		return err
	}

	runtime, err := runtimeFactory(ctx, cliRuntimeOptions{migrate: true})
	if err != nil {
		return err
	}
	defer func() {
		_ = runtime.Close()
	}()

	payload, err := getConfigPayload(ctx, runtime, resourceType, ref, showSecrets)
	if err != nil {
		return err
	}

	return writePrettyJSON(stdout, payload)
}

func runConfigUpdateCommand(
	ctx context.Context,
	resource string,
	id string,
	name string,
	patchJSON string,
	patchFile string,
	showSecrets bool,
	stdout io.Writer,
	runtimeFactory cliRuntimeFactory,
) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if runtimeFactory == nil {
		runtimeFactory = newCLIRuntime
	}

	resourceType, err := normalizeCLIConfigResourceType(resource)
	if err != nil {
		return err
	}

	ref, err := newCLIConfigReference(id, name)
	if err != nil {
		return err
	}

	patch, err := loadConfigPatch(patchJSON, patchFile)
	if err != nil {
		return err
	}

	runtime, err := runtimeFactory(ctx, cliRuntimeOptions{migrate: true})
	if err != nil {
		return err
	}
	defer func() {
		_ = runtime.Close()
	}()

	payload, err := updateConfigPayload(ctx, runtime, resourceType, ref, patch, showSecrets)
	if err != nil {
		return err
	}

	return writePrettyJSON(stdout, payload)
}

func loadConfigGroups(ctx context.Context, runtime *runtimeBundle, resource string) ([]cliConfigListGroup, error) {
	if strings.TrimSpace(resource) != "" {
		resourceType, err := normalizeCLIConfigResourceType(resource)
		if err != nil {
			return nil, err
		}
		group, err := loadConfigGroup(ctx, runtime, resourceType)
		if err != nil {
			return nil, err
		}
		return []cliConfigListGroup{group}, nil
	}

	resourceTypes := cliConfigResourceTypes()
	groups := make([]cliConfigListGroup, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		group, err := loadConfigGroup(ctx, runtime, resourceType)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func loadConfigGroup(ctx context.Context, runtime *runtimeBundle, resourceType cliConfigResourceType) (cliConfigListGroup, error) {
	switch resourceType {
	case cliConfigResourceProxmoxCluster:
		items, err := runtime.ClusterStore.List(ctx)
		if err != nil {
			return cliConfigListGroup{}, fmt.Errorf("list proxmox clusters: %w", err)
		}
		result := make([]map[string]any, 0, len(items))
		for _, item := range items {
			result = append(result, summarizeCLICluster(item))
		}
		return cliConfigListGroup{ResourceType: resourceType, Items: result}, nil
	case cliConfigResourceKubernetesCluster:
		items, err := runtime.KubernetesClusterStore.List(ctx)
		if err != nil {
			return cliConfigListGroup{}, fmt.Errorf("list kubernetes clusters: %w", err)
		}
		result := make([]map[string]any, 0, len(items))
		for _, item := range items {
			result = append(result, summarizeCLIKubernetesCluster(item))
		}
		return cliConfigListGroup{ResourceType: resourceType, Items: result}, nil
	case cliConfigResourceChannel:
		items, err := runtime.ChannelStore.List(ctx)
		if err != nil {
			return cliConfigListGroup{}, fmt.Errorf("list channels: %w", err)
		}
		result := make([]map[string]any, 0, len(items))
		for _, item := range items {
			result = append(result, summarizeCLIChannel(item))
		}
		return cliConfigListGroup{ResourceType: resourceType, Items: result}, nil
	case cliConfigResourceLLMProvider:
		items, err := runtime.LLMProviderStore.List(ctx)
		if err != nil {
			return cliConfigListGroup{}, fmt.Errorf("list llm providers: %w", err)
		}
		result := make([]map[string]any, 0, len(items))
		for _, item := range items {
			result = append(result, summarizeCLILLMProvider(item))
		}
		return cliConfigListGroup{ResourceType: resourceType, Items: result}, nil
	default:
		return cliConfigListGroup{}, fmt.Errorf("unsupported resource type %q", resourceType)
	}
}

func getConfigPayload(
	ctx context.Context,
	runtime *runtimeBundle,
	resourceType cliConfigResourceType,
	ref cliConfigReference,
	showSecrets bool,
) (map[string]any, error) {
	switch resourceType {
	case cliConfigResourceProxmoxCluster:
		cluster, err := resolveCLIClusterConfig(ctx, runtime.ClusterStore, ref)
		if err != nil {
			return nil, err
		}
		return map[string]any{"resourceType": string(resourceType), "config": cliClusterDetail(*cluster, showSecrets)}, nil
	case cliConfigResourceKubernetesCluster:
		cluster, err := resolveCLIKubernetesClusterConfig(ctx, runtime.KubernetesClusterStore, ref)
		if err != nil {
			return nil, err
		}
		return map[string]any{"resourceType": string(resourceType), "config": cliKubernetesClusterDetail(*cluster, showSecrets)}, nil
	case cliConfigResourceChannel:
		channel, err := resolveCLIChannelConfig(ctx, runtime.ChannelStore, ref)
		if err != nil {
			return nil, err
		}
		return map[string]any{"resourceType": string(resourceType), "config": cliChannelDetail(*channel, showSecrets)}, nil
	case cliConfigResourceLLMProvider:
		provider, err := resolveCLILLMProviderConfig(ctx, runtime.LLMProviderStore, ref)
		if err != nil {
			return nil, err
		}
		return map[string]any{"resourceType": string(resourceType), "config": cliLLMProviderDetail(*provider, showSecrets)}, nil
	default:
		return nil, fmt.Errorf("unsupported resource type %q", resourceType)
	}
}

func updateConfigPayload(
	ctx context.Context,
	runtime *runtimeBundle,
	resourceType cliConfigResourceType,
	ref cliConfigReference,
	patch map[string]json.RawMessage,
	showSecrets bool,
) (map[string]any, error) {
	switch resourceType {
	case cliConfigResourceProxmoxCluster:
		cluster, err := resolveCLIClusterConfig(ctx, runtime.ClusterStore, ref)
		if err != nil {
			return nil, err
		}
		if err := applyCLIClusterPatch(cluster, patch); err != nil {
			return nil, err
		}
		if err := runtime.ClusterStore.Update(ctx, cluster); err != nil {
			return nil, fmt.Errorf("update proxmox cluster %s: %w", cluster.ID, err)
		}
		return map[string]any{"status": "updated", "resourceType": string(resourceType), "config": cliClusterDetail(*cluster, showSecrets)}, nil
	case cliConfigResourceKubernetesCluster:
		cluster, err := resolveCLIKubernetesClusterConfig(ctx, runtime.KubernetesClusterStore, ref)
		if err != nil {
			return nil, err
		}
		if err := applyCLIKubernetesClusterPatch(cluster, patch); err != nil {
			return nil, err
		}
		if err := runtime.KubernetesClusterStore.Update(ctx, cluster); err != nil {
			return nil, fmt.Errorf("update kubernetes cluster %s: %w", cluster.ID, err)
		}
		return map[string]any{"status": "updated", "resourceType": string(resourceType), "config": cliKubernetesClusterDetail(*cluster, showSecrets)}, nil
	case cliConfigResourceChannel:
		channel, err := resolveCLIChannelConfig(ctx, runtime.ChannelStore, ref)
		if err != nil {
			return nil, err
		}
		if err := applyCLIChannelPatch(channel, patch); err != nil {
			return nil, err
		}
		if err := runtime.ChannelStore.Update(ctx, channel); err != nil {
			return nil, fmt.Errorf("update channel %s: %w", channel.ID, err)
		}
		return map[string]any{"status": "updated", "resourceType": string(resourceType), "config": cliChannelDetail(*channel, showSecrets)}, nil
	case cliConfigResourceLLMProvider:
		provider, err := resolveCLILLMProviderConfig(ctx, runtime.LLMProviderStore, ref)
		if err != nil {
			return nil, err
		}
		if err := applyCLILLMProviderPatch(provider, patch); err != nil {
			return nil, err
		}
		if err := runtime.LLMProviderStore.Update(ctx, provider); err != nil {
			return nil, fmt.Errorf("update llm provider %s: %w", provider.ID, err)
		}
		return map[string]any{"status": "updated", "resourceType": string(resourceType), "config": cliLLMProviderDetail(*provider, showSecrets)}, nil
	default:
		return nil, fmt.Errorf("unsupported resource type %q", resourceType)
	}
}

func renderConfigListGroups(output io.Writer, groups []cliConfigListGroup) {
	if output == nil {
		output = io.Discard
	}

	for index, group := range groups {
		if index > 0 {
			_, _ = fmt.Fprintln(output)
		}
		_, _ = fmt.Fprintf(output, "%s\n", cliConfigResourceTitle(group.ResourceType))
		if len(group.Items) == 0 {
			_, _ = fmt.Fprintf(output, "No %s found.\n", strings.ToLower(cliConfigResourceTitle(group.ResourceType)))
			continue
		}

		t := table.NewWriter()
		switch group.ResourceType {
		case cliConfigResourceProxmoxCluster:
			t.AppendHeader(table.Row{"ID", "Name", "Host", "Port", "Token ID", "Skip TLS"})
			for _, item := range group.Items {
				t.AppendRow(table.Row{stringValue(item["id"]), stringValue(item["name"]), stringValue(item["host"]), item["port"], stringValue(item["apiTokenId"]), item["skipTlsVerify"]})
			}
		case cliConfigResourceKubernetesCluster:
			t.AppendHeader(table.Row{"ID", "Name", "Source", "Context", "Namespace", "Server"})
			for _, item := range group.Items {
				t.AppendRow(table.Row{stringValue(item["id"]), stringValue(item["name"]), stringValue(item["sourceType"]), stringValue(item["contextName"]), stringValue(item["defaultNamespace"]), stringValue(item["server"])})
			}
		case cliConfigResourceChannel:
			t.AppendHeader(table.Row{"ID", "Name", "Type", "Enabled", "Connect URL"})
			for _, item := range group.Items {
				t.AppendRow(table.Row{stringValue(item["id"]), stringValue(item["name"]), stringValue(item["type"]), item["enabled"], stringValue(item["connectURL"])})
			}
		case cliConfigResourceLLMProvider:
			t.AppendHeader(table.Row{"ID", "Name", "Provider", "Model", "Default"})
			for _, item := range group.Items {
				t.AppendRow(table.Row{stringValue(item["id"]), stringValue(item["name"]), stringValue(item["providerType"]), stringValue(item["model"]), item["isDefault"]})
			}
		}
		t.SetOutputMirror(output)
		t.SetStyle(table.StyleDefault)
		t.Render()
	}
}

func newCLIConfigReference(id string, name string) (cliConfigReference, error) {
	ref := cliConfigReference{ID: strings.TrimSpace(id), Name: strings.TrimSpace(name)}
	if ref.ID == "" && ref.Name == "" {
		return cliConfigReference{}, fmt.Errorf("either --id or --name is required")
	}
	return ref, nil
}

func loadConfigPatch(patchJSON string, patchFile string) (map[string]json.RawMessage, error) {
	trimmedJSON := strings.TrimSpace(patchJSON)
	trimmedFile := strings.TrimSpace(patchFile)

	switch {
	case trimmedJSON == "" && trimmedFile == "":
		return nil, fmt.Errorf("either --patch or --patch-file is required")
	case trimmedJSON != "" && trimmedFile != "":
		return nil, fmt.Errorf("--patch and --patch-file are mutually exclusive")
	}

	raw := trimmedJSON
	if trimmedFile != "" {
		content, err := os.ReadFile(trimmedFile)
		if err != nil {
			return nil, fmt.Errorf("read patch file %s: %w", trimmedFile, err)
		}
		raw = string(content)
	}

	var patch map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &patch); err != nil {
		return nil, fmt.Errorf("parse patch JSON: %w", err)
	}
	if patch == nil {
		return nil, fmt.Errorf("patch must be a JSON object")
	}

	return patch, nil
}

func normalizeCLIConfigResourceType(value string) (cliConfigResourceType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(cliConfigResourceProxmoxCluster), "proxmoxcluster", "proxmox":
		return cliConfigResourceProxmoxCluster, nil
	case string(cliConfigResourceKubernetesCluster), "kubernetescluster", "kubernetes":
		return cliConfigResourceKubernetesCluster, nil
	case string(cliConfigResourceChannel), "channels":
		return cliConfigResourceChannel, nil
	case string(cliConfigResourceLLMProvider), "llmprovider", "provider", "providers":
		return cliConfigResourceLLMProvider, nil
	default:
		return "", fmt.Errorf("unsupported resource type %q; expected one of %s", value, strings.Join(cliConfigResourceTypeValues(), ", "))
	}
}

func cliConfigResourceTypes() []cliConfigResourceType {
	return []cliConfigResourceType{
		cliConfigResourceProxmoxCluster,
		cliConfigResourceKubernetesCluster,
		cliConfigResourceChannel,
		cliConfigResourceLLMProvider,
	}
}

func cliConfigResourceTypeValues() []string {
	resourceTypes := cliConfigResourceTypes()
	values := make([]string, 0, len(resourceTypes))
	for _, resourceType := range resourceTypes {
		values = append(values, string(resourceType))
	}
	return values
}

func cliConfigResourceTitle(resourceType cliConfigResourceType) string {
	switch resourceType {
	case cliConfigResourceProxmoxCluster:
		return "Proxmox Clusters"
	case cliConfigResourceKubernetesCluster:
		return "Kubernetes Clusters"
	case cliConfigResourceChannel:
		return "Channels"
	case cliConfigResourceLLMProvider:
		return "LLM Providers"
	default:
		return string(resourceType)
	}
}

func resolveCLIClusterConfig(ctx context.Context, store cliConfigClusterStore, ref cliConfigReference) (*models.Cluster, error) {
	if store == nil {
		return nil, fmt.Errorf("cluster store is not configured")
	}
	if ref.ID != "" {
		cluster, err := store.GetByID(ctx, ref.ID)
		if err != nil {
			return nil, fmt.Errorf("load proxmox cluster %s: %w", ref.ID, err)
		}
		if cluster == nil {
			return nil, fmt.Errorf("proxmox cluster %s was not found", ref.ID)
		}
		return cluster, nil
	}

	clusters, err := store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list proxmox clusters: %w", err)
	}

	matchID, err := findCLIConfigIDByName(clusters, ref.Name, func(cluster models.Cluster) string { return cluster.ID }, func(cluster models.Cluster) string { return cluster.Name }, "proxmox cluster")
	if err != nil {
		return nil, err
	}

	cluster, err := store.GetByID(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("load proxmox cluster %s: %w", matchID, err)
	}
	if cluster == nil {
		return nil, fmt.Errorf("proxmox cluster named %q was not found", ref.Name)
	}
	return cluster, nil
}

func resolveCLIKubernetesClusterConfig(ctx context.Context, store cliConfigKubernetesClusterStore, ref cliConfigReference) (*models.KubernetesCluster, error) {
	if store == nil {
		return nil, fmt.Errorf("kubernetes cluster store is not configured")
	}
	if ref.ID != "" {
		cluster, err := store.GetByID(ctx, ref.ID)
		if err != nil {
			return nil, fmt.Errorf("load kubernetes cluster %s: %w", ref.ID, err)
		}
		if cluster == nil {
			return nil, fmt.Errorf("kubernetes cluster %s was not found", ref.ID)
		}
		return cluster, nil
	}

	clusters, err := store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list kubernetes clusters: %w", err)
	}

	matchID, err := findCLIConfigIDByName(clusters, ref.Name, func(cluster models.KubernetesCluster) string { return cluster.ID }, func(cluster models.KubernetesCluster) string { return cluster.Name }, "kubernetes cluster")
	if err != nil {
		return nil, err
	}

	cluster, err := store.GetByID(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("load kubernetes cluster %s: %w", matchID, err)
	}
	if cluster == nil {
		return nil, fmt.Errorf("kubernetes cluster named %q was not found", ref.Name)
	}
	return cluster, nil
}

func resolveCLIChannelConfig(ctx context.Context, store cliConfigChannelStore, ref cliConfigReference) (*models.Channel, error) {
	if store == nil {
		return nil, fmt.Errorf("channel store is not configured")
	}
	if ref.ID != "" {
		channel, err := store.GetByID(ctx, ref.ID)
		if err != nil {
			return nil, fmt.Errorf("load channel %s: %w", ref.ID, err)
		}
		if channel == nil {
			return nil, fmt.Errorf("channel %s was not found", ref.ID)
		}
		return channel, nil
	}

	channels, err := store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}

	matchID, err := findCLIConfigIDByName(channels, ref.Name, func(channel models.Channel) string { return channel.ID }, func(channel models.Channel) string { return channel.Name }, "channel")
	if err != nil {
		return nil, err
	}

	channel, err := store.GetByID(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("load channel %s: %w", matchID, err)
	}
	if channel == nil {
		return nil, fmt.Errorf("channel named %q was not found", ref.Name)
	}
	return channel, nil
}

func resolveCLILLMProviderConfig(ctx context.Context, store cliConfigLLMProviderStore, ref cliConfigReference) (*models.LLMProvider, error) {
	if store == nil {
		return nil, fmt.Errorf("llm provider store is not configured")
	}
	if ref.ID != "" {
		provider, err := store.GetByID(ctx, ref.ID)
		if err != nil {
			return nil, fmt.Errorf("load llm provider %s: %w", ref.ID, err)
		}
		if provider == nil {
			return nil, fmt.Errorf("llm provider %s was not found", ref.ID)
		}
		return provider, nil
	}

	providers, err := store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list llm providers: %w", err)
	}

	matchID, err := findCLIConfigIDByName(providers, ref.Name, func(provider models.LLMProvider) string { return provider.ID }, func(provider models.LLMProvider) string { return provider.Name }, "llm provider")
	if err != nil {
		return nil, err
	}

	provider, err := store.GetByID(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("load llm provider %s: %w", matchID, err)
	}
	if provider == nil {
		return nil, fmt.Errorf("llm provider named %q was not found", ref.Name)
	}
	return provider, nil
}

func findCLIConfigIDByName[T any](items []T, name string, idFn func(T) string, nameFn func(T) string, kind string) (string, error) {
	trimmedName := strings.TrimSpace(name)
	matches := make([]string, 0, 1)
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(nameFn(item)), trimmedName) {
			matches = append(matches, idFn(item))
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%s named %q was not found", kind, name)
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("multiple %ss named %q matched ids: %s", kind, name, strings.Join(matches, ", "))
	}
}

func applyCLIClusterPatch(cluster *models.Cluster, patch map[string]json.RawMessage) error {
	if cluster == nil {
		return fmt.Errorf("cluster is required")
	}

	if value, present, err := parseOptionalStringField(patch, "name"); err != nil {
		return err
	} else if present {
		cluster.Name = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "host"); err != nil {
		return err
	} else if present {
		cluster.Host = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalIntField(patch, "port"); err != nil {
		return err
	} else if present {
		cluster.Port = value
	}
	if value, present, err := parseOptionalStringField(patch, "apiTokenId"); err != nil {
		return err
	} else if present {
		cluster.APITokenID = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "apiTokenSecret"); err != nil {
		return err
	} else if present {
		cluster.APITokenSecret = value
	}
	if value, present, err := parseOptionalBoolField(patch, "skipTlsVerify"); err != nil {
		return err
	} else if present {
		cluster.SkipTLSVerify = value
	}

	return nil
}

func applyCLIKubernetesClusterPatch(cluster *models.KubernetesCluster, patch map[string]json.RawMessage) error {
	if cluster == nil {
		return fmt.Errorf("kubernetes cluster is required")
	}

	if value, present, err := parseOptionalStringField(patch, "name"); err != nil {
		return err
	} else if present {
		cluster.Name = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "sourceType"); err != nil {
		return err
	} else if present {
		cluster.SourceType = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "kubeconfig"); err != nil {
		return err
	} else if present {
		cluster.Kubeconfig = value
	}
	if value, present, err := parseOptionalStringField(patch, "contextName"); err != nil {
		return err
	} else if present {
		cluster.ContextName = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "defaultNamespace"); err != nil {
		return err
	} else if present {
		cluster.DefaultNamespace = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "server"); err != nil {
		return err
	} else if present {
		cluster.Server = strings.TrimSpace(value)
	}

	return nil
}

func applyCLIChannelPatch(channel *models.Channel, patch map[string]json.RawMessage) error {
	if channel == nil {
		return fmt.Errorf("channel is required")
	}

	if value, present, err := parseOptionalStringField(patch, "name"); err != nil {
		return err
	} else if present {
		channel.Name = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "type"); err != nil {
		return err
	} else if present {
		channel.Type = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalJSONDocumentField(patch, "config"); err != nil {
		return err
	} else if present {
		channel.Config = value
	}
	if value, present, err := parseOptionalStringField(patch, "welcomeMessage"); err != nil {
		return err
	} else if present {
		channel.WelcomeMessage = value
	}
	if value, present, err := parseOptionalStringPointerField(patch, "connectURL"); err != nil {
		return err
	} else if present {
		channel.ConnectURL = value
	}
	if value, present, err := parseOptionalBoolField(patch, "enabled"); err != nil {
		return err
	} else if present {
		channel.Enabled = value
	}
	if value, present, err := parseOptionalStringPointerField(patch, "state"); err != nil {
		return err
	} else if present {
		channel.State = value
	}

	return nil
}

func applyCLILLMProviderPatch(provider *models.LLMProvider, patch map[string]json.RawMessage) error {
	if provider == nil {
		return fmt.Errorf("llm provider is required")
	}

	if value, present, err := parseOptionalStringField(patch, "name"); err != nil {
		return err
	} else if present {
		provider.Name = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "providerType"); err != nil {
		return err
	} else if present {
		provider.ProviderType = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalStringField(patch, "apiKey"); err != nil {
		return err
	} else if present {
		provider.APIKey = value
	}
	if value, present, err := parseOptionalStringPointerField(patch, "baseURL"); err != nil {
		return err
	} else if present {
		provider.BaseURL = value
	}
	if value, present, err := parseOptionalStringField(patch, "model"); err != nil {
		return err
	} else if present {
		provider.Model = strings.TrimSpace(value)
	}
	if value, present, err := parseOptionalJSONDocumentField(patch, "config"); err != nil {
		return err
	} else if present {
		provider.Config = value
	}
	if value, present, err := parseOptionalBoolField(patch, "isDefault"); err != nil {
		return err
	} else if present {
		provider.IsDefault = value
	}

	return nil
}

func summarizeCLICluster(cluster models.Cluster) map[string]any {
	return map[string]any{
		"id":            cluster.ID,
		"name":          cluster.Name,
		"host":          cluster.Host,
		"port":          cluster.Port,
		"apiTokenId":    cluster.APITokenID,
		"skipTlsVerify": cluster.SkipTLSVerify,
		"createdAt":     cluster.CreatedAt,
		"updatedAt":     cluster.UpdatedAt,
	}
}

func cliClusterDetail(cluster models.Cluster, showSecrets bool) map[string]any {
	detail := summarizeCLICluster(cluster)
	if showSecrets {
		detail["apiTokenSecret"] = cluster.APITokenSecret
	} else {
		detail["apiTokenSecretConfigured"] = strings.TrimSpace(cluster.APITokenSecret) != ""
	}
	return detail
}

func summarizeCLIKubernetesCluster(cluster models.KubernetesCluster) map[string]any {
	return map[string]any{
		"id":               cluster.ID,
		"name":             cluster.Name,
		"sourceType":       cluster.SourceType,
		"contextName":      cluster.ContextName,
		"defaultNamespace": cluster.DefaultNamespace,
		"server":           cluster.Server,
		"createdAt":        cluster.CreatedAt,
		"updatedAt":        cluster.UpdatedAt,
	}
}

func cliKubernetesClusterDetail(cluster models.KubernetesCluster, showSecrets bool) map[string]any {
	detail := summarizeCLIKubernetesCluster(cluster)
	if showSecrets {
		detail["kubeconfig"] = cluster.Kubeconfig
	} else {
		detail["kubeconfigConfigured"] = strings.TrimSpace(cluster.Kubeconfig) != ""
	}
	return detail
}

func summarizeCLIChannel(channel models.Channel) map[string]any {
	return map[string]any{
		"id":             channel.ID,
		"name":           channel.Name,
		"type":           channel.Type,
		"welcomeMessage": channel.WelcomeMessage,
		"connectURL":     channel.ConnectURL,
		"enabled":        channel.Enabled,
		"createdAt":      channel.CreatedAt,
		"updatedAt":      channel.UpdatedAt,
	}
}

func cliChannelDetail(channel models.Channel, showSecrets bool) map[string]any {
	detail := summarizeCLIChannel(channel)
	detail["state"] = channel.State
	if showSecrets {
		detail["config"] = decodeJSONDocument(channel.Config)
	} else {
		detail["configSummary"] = summarizeJSONDocument(channel.Config)
	}
	return detail
}

func summarizeCLILLMProvider(provider models.LLMProvider) map[string]any {
	return map[string]any{
		"id":           provider.ID,
		"name":         provider.Name,
		"providerType": provider.ProviderType,
		"baseURL":      provider.BaseURL,
		"model":        provider.Model,
		"isDefault":    provider.IsDefault,
		"createdAt":    provider.CreatedAt,
		"updatedAt":    provider.UpdatedAt,
	}
}

func cliLLMProviderDetail(provider models.LLMProvider, showSecrets bool) map[string]any {
	detail := summarizeCLILLMProvider(provider)
	if showSecrets {
		detail["apiKey"] = provider.APIKey
		detail["config"] = decodeJSONDocument(provider.Config)
	} else {
		detail["apiKeyConfigured"] = strings.TrimSpace(provider.APIKey) != ""
		detail["configSummary"] = summarizeJSONDocument(provider.Config)
	}
	return detail
}

func decodeJSONDocument(raw *string) any {
	if raw == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return trimmed
	}
	return decoded
}

func summarizeJSONDocument(raw *string) any {
	if raw == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return map[string]any{"type": "string", "length": len(trimmed)}
	}

	return summarizeJSONValue(decoded)
}

func summarizeJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		fieldTypes := make(map[string]string, len(typed))
		for key, child := range typed {
			keys = append(keys, key)
			fieldTypes[key] = jsonValueType(child)
		}
		sort.Strings(keys)
		return map[string]any{"type": "object", "keys": keys, "fieldTypes": fieldTypes}
	case []any:
		itemTypes := make([]string, 0)
		seen := make(map[string]struct{})
		for _, item := range typed {
			itemType := jsonValueType(item)
			if _, exists := seen[itemType]; exists {
				continue
			}
			seen[itemType] = struct{}{}
			itemTypes = append(itemTypes, itemType)
		}
		sort.Strings(itemTypes)
		return map[string]any{"type": "array", "length": len(typed), "itemTypes": itemTypes}
	default:
		return map[string]any{"type": jsonValueType(value)}
	}
}

func jsonValueType(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func parseOptionalStringField(fields map[string]json.RawMessage, key string) (string, bool, error) {
	raw, ok := fields[key]
	if !ok {
		return "", false, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", true, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, true, nil
}

func parseOptionalStringPointerField(fields map[string]json.RawMessage, key string) (*string, bool, error) {
	raw, ok := fields[key]
	if !ok {
		return nil, false, nil
	}
	if strings.TrimSpace(string(raw)) == "null" {
		return nil, true, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, true, fmt.Errorf("parse %s: %w", key, err)
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, true, nil
	}
	return &trimmed, true, nil
}

func parseOptionalBoolField(fields map[string]json.RawMessage, key string) (bool, bool, error) {
	raw, ok := fields[key]
	if !ok {
		return false, false, nil
	}

	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, true, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, true, nil
}

func parseOptionalIntField(fields map[string]json.RawMessage, key string) (int, bool, error) {
	raw, ok := fields[key]
	if !ok {
		return 0, false, nil
	}

	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, true, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, true, nil
}

func parseOptionalJSONDocumentField(fields map[string]json.RawMessage, key string) (*string, bool, error) {
	raw, ok := fields[key]
	if !ok {
		return nil, false, nil
	}
	if strings.TrimSpace(string(raw)) == "null" {
		return nil, true, nil
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, true, nil
	}

	if strings.HasPrefix(trimmed, "\"") {
		var embedded string
		if err := json.Unmarshal(raw, &embedded); err != nil {
			return nil, true, fmt.Errorf("parse %s: %w", key, err)
		}
		embedded = strings.TrimSpace(embedded)
		if embedded == "" {
			return nil, true, nil
		}
		if !json.Valid([]byte(embedded)) {
			return nil, true, fmt.Errorf("%s must be a JSON object, array, or a string containing valid JSON", key)
		}
		compacted, err := compactJSONString(embedded)
		if err != nil {
			return nil, true, fmt.Errorf("normalize %s: %w", key, err)
		}
		return &compacted, true, nil
	}

	compacted, err := compactJSONString(trimmed)
	if err != nil {
		return nil, true, fmt.Errorf("normalize %s: %w", key, err)
	}
	return &compacted, true, nil
}

func compactJSONString(value string) (string, error) {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, []byte(value)); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func writePrettyJSON(output io.Writer, payload any) error {
	if output == nil {
		output = io.Discard
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case *string:
		if typed == nil {
			return ""
		}
		return *typed
	default:
		return fmt.Sprint(value)
	}
}
