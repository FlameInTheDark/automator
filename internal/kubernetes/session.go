package kubernetes

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Session exposes initialized Kubernetes clients for a normalized cluster.
type Session struct {
	rawConfig    clientcmdapi.Config
	clientConfig clientcmd.ClientConfig
	restConfig   *rest.Config
	discovery    discovery.CachedDiscoveryInterface
	mapper       meta.RESTMapper
	dynamic      dynamic.Interface
	clientset    kubernetes.Interface
	contextName  string
	namespace    string
	server       string
	restGetter   *restClientGetter
}

// NewSessionFromKubeconfig creates a ready-to-use session from kubeconfig data.
func NewSessionFromKubeconfig(kubeconfig string, contextName string) (*Session, error) {
	rawConfig, effectiveContext, err := loadRawConfig(kubeconfig, contextName)
	if err != nil {
		return nil, err
	}

	overrides := &clientcmd.ConfigOverrides{}
	if effectiveContext != "" {
		overrides.CurrentContext = effectiveContext
	}

	clientConfig := clientcmd.NewDefaultClientConfig(rawConfig, overrides)
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}
	cachedDiscovery := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create typed client: %w", err)
	}

	namespace, _, err := clientConfig.Namespace()
	if err != nil || strings.TrimSpace(namespace) == "" {
		namespace = "default"
	}

	clusterName := rawConfig.Contexts[effectiveContext].Cluster
	server := ""
	if cluster, ok := rawConfig.Clusters[clusterName]; ok {
		server = strings.TrimSpace(cluster.Server)
	}

	restGetter := &restClientGetter{
		clientConfig: clientConfig,
		discovery:    cachedDiscovery,
		mapper:       mapper,
	}

	return &Session{
		rawConfig:    rawConfig,
		clientConfig: clientConfig,
		restConfig:   restConfig,
		discovery:    cachedDiscovery,
		mapper:       mapper,
		dynamic:      dynamicClient,
		clientset:    clientset,
		contextName:  effectiveContext,
		namespace:    namespace,
		server:       server,
		restGetter:   restGetter,
	}, nil
}

// RestConfig returns the session's rest.Config.
func (s *Session) RestConfig() *rest.Config {
	return s.restConfig
}

// Discovery returns the cached discovery client.
func (s *Session) Discovery() discovery.CachedDiscoveryInterface {
	return s.discovery
}

// Mapper returns the REST mapper for the session.
func (s *Session) Mapper() meta.RESTMapper {
	return s.mapper
}

// Dynamic returns the dynamic client.
func (s *Session) Dynamic() dynamic.Interface {
	return s.dynamic
}

// Clientset returns the typed clientset.
func (s *Session) Clientset() kubernetes.Interface {
	return s.clientset
}

// ContextName returns the effective kubeconfig context.
func (s *Session) ContextName() string {
	return s.contextName
}

// Namespace returns the default namespace resolved from kubeconfig.
func (s *Session) Namespace() string {
	return s.namespace
}

// Server returns the current API server URL.
func (s *Session) Server() string {
	return s.server
}

// NewBuilder exposes a kubectl-compatible resource builder bound to this session.
func (s *Session) NewBuilder() *resource.Builder {
	return resource.NewBuilder(s.restGetter).Unstructured()
}

type restClientGetter struct {
	clientConfig clientcmd.ClientConfig
	discovery    discovery.CachedDiscoveryInterface
	mapper       meta.RESTMapper
}

func (g *restClientGetter) ToRESTConfig() (*rest.Config, error) {
	return g.clientConfig.ClientConfig()
}

func (g *restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return g.discovery, nil
}

func (g *restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return g.mapper, nil
}

func (g *restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return g.clientConfig
}
