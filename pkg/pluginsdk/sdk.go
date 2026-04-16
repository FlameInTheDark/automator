package pluginsdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/FlameInTheDark/emerald/pkg/pluginapi"
	"github.com/FlameInTheDark/emerald/pkg/pluginrpc"
)

const PluginName = "emerald"

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "EMERALD_PLUGIN",
	MagicCookieValue: "emerald-plugin",
}

type Plugin struct {
	plugin.NetRPCUnsupportedPlugin
	Impl pluginapi.Plugin
}

func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, server *grpc.Server) error {
	pluginrpc.RegisterEmeraldPluginServer(server, &grpcServer{impl: p.Impl})
	return nil
}

func (p *Plugin) GRPCClient(ctx context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{
		ctx:    ctx,
		client: pluginrpc.NewEmeraldPluginClient(conn),
	}, nil
}

func PluginMap(impl pluginapi.Plugin) plugin.PluginSet {
	return plugin.PluginSet{
		PluginName: &Plugin{Impl: impl},
	}
}

func Serve(impl pluginapi.Plugin) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         PluginMap(impl),
		GRPCServer:      plugin.DefaultGRPCServer,
	})
}

func NewClientConfig(cmd *exec.Cmd) *plugin.ClientConfig {
	return &plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		Plugins:          PluginMap(nil),
		Cmd:              cmd,
		Managed:          true,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           hclog.NewNullLogger(),
	}
}

type grpcServer struct {
	pluginrpc.UnimplementedEmeraldPluginServer
	impl pluginapi.Plugin
}

func (s *grpcServer) Describe(ctx context.Context, _ *pluginrpc.DescribeRequest) (*pluginrpc.DescribeResponse, error) {
	info, err := s.impl.Describe(ctx)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal plugin info: %w", err)
	}

	return &pluginrpc.DescribeResponse{PluginInfoJson: payload}, nil
}

func (s *grpcServer) ValidateConfig(ctx context.Context, req *pluginrpc.ValidateConfigRequest) (*pluginrpc.ValidateConfigResponse, error) {
	if err := s.impl.ValidateConfig(ctx, req.GetNodeId(), req.GetConfigJson()); err != nil {
		return nil, err
	}

	return &pluginrpc.ValidateConfigResponse{}, nil
}

func (s *grpcServer) ExecuteAction(ctx context.Context, req *pluginrpc.ExecuteActionRequest) (*pluginrpc.ExecuteActionResponse, error) {
	input, err := decodeInput(req.GetInputJson())
	if err != nil {
		return nil, err
	}

	output, err := s.impl.ExecuteAction(ctx, req.GetNodeId(), req.GetConfigJson(), input)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("marshal action output: %w", err)
	}

	return &pluginrpc.ExecuteActionResponse{OutputJson: payload}, nil
}

func (s *grpcServer) ToolDefinition(ctx context.Context, req *pluginrpc.ToolDefinitionRequest) (*pluginrpc.ToolDefinitionResponse, error) {
	var meta pluginapi.ToolNodeMetadata
	if err := unmarshalJSONPayload(req.GetMetaJson(), &meta); err != nil {
		return nil, fmt.Errorf("decode tool metadata: %w", err)
	}

	definition, err := s.impl.ToolDefinition(ctx, req.GetNodeId(), meta, req.GetConfigJson())
	if err != nil {
		return nil, err
	}
	if definition == nil {
		return nil, fmt.Errorf("tool definition is required")
	}

	payload, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("marshal tool definition: %w", err)
	}

	return &pluginrpc.ToolDefinitionResponse{DefinitionJson: payload}, nil
}

func (s *grpcServer) ExecuteTool(ctx context.Context, req *pluginrpc.ExecuteToolRequest) (*pluginrpc.ExecuteToolResponse, error) {
	input, err := decodeInput(req.GetInputJson())
	if err != nil {
		return nil, err
	}

	result, err := s.impl.ExecuteTool(ctx, req.GetNodeId(), req.GetConfigJson(), req.GetArgsJson(), input)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal tool result: %w", err)
	}

	return &pluginrpc.ExecuteToolResponse{ResultJson: payload}, nil
}

func (s *grpcServer) TriggerRuntime(stream pluginrpc.EmeraldPlugin_TriggerRuntimeServer) error {
	runtime, err := s.impl.OpenTriggerRuntime(stream.Context())
	if err != nil {
		return err
	}
	defer func() {
		_ = runtime.Close()
	}()

	recvErrCh := make(chan error, 1)
	go func() {
		defer close(recvErrCh)

		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
					return
				}
				_ = runtime.Close()
				recvErrCh <- err
				return
			}

			var snapshot pluginapi.TriggerSubscriptionSnapshot
			if err := unmarshalOptionalJSONPayload(msg.GetSnapshotJson(), &snapshot); err != nil {
				_ = runtime.Close()
				recvErrCh <- fmt.Errorf("decode trigger snapshot: %w", err)
				return
			}

			if err := runtime.SendSnapshot(stream.Context(), snapshot); err != nil {
				_ = runtime.Close()
				recvErrCh <- err
				return
			}
		}
	}()

	for {
		event, err := runtime.Recv(stream.Context())
		if err != nil {
			if recvErr, ok := <-recvErrCh; ok && recvErr != nil {
				return recvErr
			}
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if event == nil {
			continue
		}

		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal trigger event: %w", err)
		}
		if err := stream.Send(&pluginrpc.TriggerRuntimeServerMessage{EventJson: payload}); err != nil {
			return err
		}
	}
}

type grpcClient struct {
	ctx    context.Context
	client pluginrpc.EmeraldPluginClient
}

func (c *grpcClient) Describe(ctx context.Context) (pluginapi.PluginInfo, error) {
	resp, err := c.client.Describe(c.callContext(ctx), &pluginrpc.DescribeRequest{})
	if err != nil {
		return pluginapi.PluginInfo{}, err
	}

	var info pluginapi.PluginInfo
	if err := unmarshalJSONPayload(resp.GetPluginInfoJson(), &info); err != nil {
		return pluginapi.PluginInfo{}, fmt.Errorf("decode plugin info: %w", err)
	}

	return info, nil
}

func (c *grpcClient) ValidateConfig(ctx context.Context, nodeID string, config json.RawMessage) error {
	_, err := c.client.ValidateConfig(c.callContext(ctx), &pluginrpc.ValidateConfigRequest{
		NodeId:     nodeID,
		ConfigJson: normalizeJSONPayload(config),
	})
	return err
}

func (c *grpcClient) ExecuteAction(ctx context.Context, nodeID string, config json.RawMessage, input map[string]any) (any, error) {
	inputPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode action input: %w", err)
	}

	resp, err := c.client.ExecuteAction(c.callContext(ctx), &pluginrpc.ExecuteActionRequest{
		NodeId:     nodeID,
		ConfigJson: normalizeJSONPayload(config),
		InputJson:  inputPayload,
	})
	if err != nil {
		return nil, err
	}

	return decodeResult(resp.GetOutputJson())
}

func (c *grpcClient) ToolDefinition(ctx context.Context, nodeID string, meta pluginapi.ToolNodeMetadata, config json.RawMessage) (*pluginapi.ToolDefinition, error) {
	metaPayload, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("encode tool metadata: %w", err)
	}

	resp, err := c.client.ToolDefinition(c.callContext(ctx), &pluginrpc.ToolDefinitionRequest{
		NodeId:     nodeID,
		MetaJson:   metaPayload,
		ConfigJson: normalizeJSONPayload(config),
	})
	if err != nil {
		return nil, err
	}

	var definition pluginapi.ToolDefinition
	if err := unmarshalJSONPayload(resp.GetDefinitionJson(), &definition); err != nil {
		return nil, fmt.Errorf("decode tool definition: %w", err)
	}

	return &definition, nil
}

func (c *grpcClient) ExecuteTool(ctx context.Context, nodeID string, config json.RawMessage, args json.RawMessage, input map[string]any) (any, error) {
	inputPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode tool input: %w", err)
	}

	resp, err := c.client.ExecuteTool(c.callContext(ctx), &pluginrpc.ExecuteToolRequest{
		NodeId:     nodeID,
		ConfigJson: normalizeJSONPayload(config),
		ArgsJson:   normalizeJSONPayload(args),
		InputJson:  inputPayload,
	})
	if err != nil {
		return nil, err
	}

	return decodeResult(resp.GetResultJson())
}

func (c *grpcClient) OpenTriggerRuntime(ctx context.Context) (pluginapi.TriggerRuntime, error) {
	stream, err := c.client.TriggerRuntime(c.callContext(ctx))
	if err != nil {
		return nil, err
	}

	return &remoteTriggerRuntime{
		stream: stream,
	}, nil
}

func (c *grpcClient) callContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

func decodeInput(payload []byte) (map[string]any, error) {
	if len(payload) == 0 {
		return map[string]any{}, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode input payload: %w", err)
	}
	if decoded == nil {
		return map[string]any{}, nil
	}

	return decoded, nil
}

func decodeResult(payload []byte) (any, error) {
	if len(payload) == 0 {
		return map[string]any{}, nil
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode result payload: %w", err)
	}

	return decoded, nil
}

func normalizeJSONPayload(payload []byte) []byte {
	if len(payload) == 0 {
		return []byte("{}")
	}
	return payload
}

func unmarshalJSONPayload[T any](payload []byte, target *T) error {
	if len(payload) == 0 {
		return fmt.Errorf("payload is required")
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return err
	}
	return nil
}

func unmarshalOptionalJSONPayload[T any](payload []byte, target *T) error {
	if len(payload) == 0 {
		var zero T
		*target = zero
		return nil
	}
	return json.Unmarshal(payload, target)
}

type remoteTriggerRuntime struct {
	stream    pluginrpc.EmeraldPlugin_TriggerRuntimeClient
	sendMu    sync.Mutex
	closeOnce sync.Once
	closeErr  error
}

func (r *remoteTriggerRuntime) SendSnapshot(_ context.Context, snapshot pluginapi.TriggerSubscriptionSnapshot) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encode trigger snapshot: %w", err)
	}

	r.sendMu.Lock()
	defer r.sendMu.Unlock()

	return r.stream.Send(&pluginrpc.TriggerRuntimeClientMessage{SnapshotJson: payload})
}

func (r *remoteTriggerRuntime) Recv(_ context.Context) (*pluginapi.TriggerEvent, error) {
	resp, err := r.stream.Recv()
	if err != nil {
		return nil, err
	}

	var event pluginapi.TriggerEvent
	if err := unmarshalJSONPayload(resp.GetEventJson(), &event); err != nil {
		return nil, fmt.Errorf("decode trigger event: %w", err)
	}

	return &event, nil
}

func (r *remoteTriggerRuntime) Close() error {
	r.closeOnce.Do(func() {
		r.closeErr = r.stream.CloseSend()
	})

	return r.closeErr
}
