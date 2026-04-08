package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// PodLogs collects pod logs.
func (s *Session) PodLogs(ctx context.Context, opts PodLogOptions) (string, ResourceReference, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return "", ResourceReference{}, fmt.Errorf("name is required")
	}

	namespace := strings.TrimSpace(opts.Namespace)
	if namespace == "" {
		namespace = s.Namespace()
	}
	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}

	request := s.clientset.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{
		Container:    strings.TrimSpace(opts.Container),
		TailLines:    optionalInt64(opts.TailLines),
		SinceSeconds: optionalInt64(opts.SinceSeconds),
		Timestamps:   opts.Timestamps,
		Previous:     opts.Previous,
	})

	stream, err := request.Stream(ctx)
	if err != nil {
		return "", ResourceReference{}, fmt.Errorf("stream pod logs: %w", err)
	}
	defer func() {
		_ = stream.Close()
	}()

	data, err := io.ReadAll(stream)
	if err != nil {
		return "", ResourceReference{}, fmt.Errorf("read pod logs: %w", err)
	}

	return string(data), ResourceReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Resource:   "pods",
		Namespace:  namespace,
		Name:       name,
	}, nil
}

// PodExec runs a non-interactive command inside a pod.
func (s *Session) PodExec(ctx context.Context, opts PodExecOptions) (*PodExecResult, ResourceReference, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return nil, ResourceReference{}, fmt.Errorf("name is required")
	}
	if len(opts.Command) == 0 {
		return nil, ResourceReference{}, fmt.Errorf("command is required")
	}

	namespace := strings.TrimSpace(opts.Namespace)
	if namespace == "" {
		namespace = s.Namespace()
	}
	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}

	request := s.clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: strings.TrimSpace(opts.Container),
			Command:   opts.Command,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(s.restConfig, "POST", request.URL())
	if err != nil {
		return nil, ResourceReference{}, fmt.Errorf("create exec stream: %w", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	}); err != nil {
		return nil, ResourceReference{}, fmt.Errorf("exec in pod: %w", err)
	}

	return &PodExecResult{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, ResourceReference{
			APIVersion: "v1",
			Kind:       "Pod",
			Resource:   "pods",
			Namespace:  namespace,
			Name:       name,
		}, nil
}

// ListEvents returns cluster events for the requested namespace.
func (s *Session) ListEvents(ctx context.Context, opts EventOptions) ([]map[string]any, ResourceReference, error) {
	namespace := strings.TrimSpace(opts.Namespace)
	if namespace == "" {
		namespace = s.Namespace()
	}

	fieldSelectors := make([]string, 0, 4)
	if selector := strings.TrimSpace(opts.FieldSelector); selector != "" {
		fieldSelectors = append(fieldSelectors, selector)
	}
	if value := strings.TrimSpace(opts.InvolvedObjectName); value != "" {
		fieldSelectors = append(fieldSelectors, "involvedObject.name="+value)
	}
	if value := strings.TrimSpace(opts.InvolvedObjectKind); value != "" {
		fieldSelectors = append(fieldSelectors, "involvedObject.kind="+value)
	}
	if value := strings.TrimSpace(opts.InvolvedObjectUID); value != "" {
		fieldSelectors = append(fieldSelectors, "involvedObject.uid="+value)
	}

	listOptions := metav1.ListOptions{
		FieldSelector: strings.Join(fieldSelectors, ","),
	}
	if opts.Limit > 0 {
		listOptions.Limit = opts.Limit
	}

	list, err := s.clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, ResourceReference{}, fmt.Errorf("list events: %w", err)
	}

	events := make([]map[string]any, 0, len(list.Items))
	for _, event := range list.Items {
		data, err := runtimeToMap(event)
		if err != nil {
			return nil, ResourceReference{}, err
		}
		events = append(events, data)
	}

	return events, ResourceReference{
		APIVersion: "v1",
		Kind:       "Event",
		Resource:   "events",
		Namespace:  namespace,
	}, nil
}

func runtimeToMap(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime object: %w", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("unmarshal runtime object: %w", err)
	}

	return sanitizeObject(decoded), nil
}

func optionalInt64(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}
