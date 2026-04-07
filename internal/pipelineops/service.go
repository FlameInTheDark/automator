package pipelineops

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FlameInTheDark/automator/internal/db/models"
)

const (
	StatusDraft    = "draft"
	StatusActive   = "active"
	StatusArchived = "archived"
)

type Store interface {
	List(ctx context.Context) ([]models.Pipeline, error)
	GetByID(ctx context.Context, id string) (*models.Pipeline, error)
	Create(ctx context.Context, pipeline *models.Pipeline) error
	Update(ctx context.Context, pipeline *models.Pipeline) error
	Delete(ctx context.Context, id string) error
}

type Reloader interface {
	Reload(ctx context.Context) error
}

type Service struct {
	store    Store
	reloader Reloader
}

type Reference struct {
	ID   string
	Name string
}

type flowNode struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type flowEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
}

func NewService(store Store, reloader Reloader) *Service {
	return &Service{
		store:    store,
		reloader: reloader,
	}
}

func (s *Service) List(ctx context.Context, ref Reference) ([]models.Pipeline, error) {
	if s.store == nil {
		return nil, fmt.Errorf("pipeline store is not configured")
	}

	pipelines, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}

	return FilterPipelines(pipelines, ref), nil
}

func (s *Service) Resolve(ctx context.Context, ref Reference) (*models.Pipeline, error) {
	if s.store == nil {
		return nil, fmt.Errorf("pipeline store is not configured")
	}

	if pipelineID := strings.TrimSpace(ref.ID); pipelineID != "" {
		pipelineModel, err := s.store.GetByID(ctx, pipelineID)
		if err != nil {
			return nil, fmt.Errorf("load pipeline %s: %w", pipelineID, err)
		}
		return pipelineModel, nil
	}

	pipelineName := strings.TrimSpace(ref.Name)
	if pipelineName == "" {
		return nil, fmt.Errorf("pipelineId or pipelineName is required")
	}

	pipelines, err := s.List(ctx, Reference{Name: pipelineName})
	if err != nil {
		return nil, err
	}
	if len(pipelines) == 0 {
		return nil, fmt.Errorf("pipeline named %q was not found", pipelineName)
	}
	if len(pipelines) > 1 {
		return nil, fmt.Errorf("multiple pipelines named %q were found; use pipelineId instead", pipelineName)
	}

	pipelineCopy := pipelines[0]
	return &pipelineCopy, nil
}

func (s *Service) Create(ctx context.Context, pipelineModel *models.Pipeline) error {
	if s.store == nil {
		return fmt.Errorf("pipeline store is not configured")
	}
	if err := NormalizePipeline(pipelineModel); err != nil {
		return err
	}
	if err := s.store.Create(ctx, pipelineModel); err != nil {
		return fmt.Errorf("create pipeline: %w", err)
	}

	s.reload(ctx)
	return nil
}

func (s *Service) Update(ctx context.Context, pipelineModel *models.Pipeline) error {
	if s.store == nil {
		return fmt.Errorf("pipeline store is not configured")
	}
	if err := NormalizePipeline(pipelineModel); err != nil {
		return err
	}
	if err := s.store.Update(ctx, pipelineModel); err != nil {
		return fmt.Errorf("update pipeline: %w", err)
	}

	s.reload(ctx)
	return nil
}

func (s *Service) Delete(ctx context.Context, ref Reference) (*models.Pipeline, error) {
	if s.store == nil {
		return nil, fmt.Errorf("pipeline store is not configured")
	}

	pipelineModel, err := s.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	if err := s.store.Delete(ctx, pipelineModel.ID); err != nil {
		return nil, fmt.Errorf("delete pipeline %s: %w", pipelineModel.ID, err)
	}

	s.reload(ctx)
	return pipelineModel, nil
}

func FilterPipelines(pipelines []models.Pipeline, ref Reference) []models.Pipeline {
	pipelineID := strings.TrimSpace(ref.ID)
	pipelineName := strings.TrimSpace(ref.Name)
	if pipelineID == "" && pipelineName == "" {
		return pipelines
	}

	filtered := make([]models.Pipeline, 0, len(pipelines))
	for _, pipelineModel := range pipelines {
		if pipelineID != "" && pipelineModel.ID != pipelineID {
			continue
		}
		if pipelineName != "" && !strings.EqualFold(pipelineModel.Name, pipelineName) {
			continue
		}
		filtered = append(filtered, pipelineModel)
	}

	return filtered
}

func NormalizePipeline(pipelineModel *models.Pipeline) error {
	if pipelineModel == nil {
		return fmt.Errorf("pipeline is required")
	}
	if strings.TrimSpace(pipelineModel.Name) == "" {
		return fmt.Errorf("name is required")
	}

	pipelineModel.Name = strings.TrimSpace(pipelineModel.Name)
	pipelineModel.Status = normalizeStatus(pipelineModel.Status)
	if pipelineModel.Status == "" {
		pipelineModel.Status = StatusDraft
	}
	if !isValidStatus(pipelineModel.Status) {
		return fmt.Errorf("status must be one of %s, %s, or %s", StatusDraft, StatusActive, StatusArchived)
	}

	if strings.TrimSpace(pipelineModel.Nodes) == "" {
		pipelineModel.Nodes = "[]"
	}
	if strings.TrimSpace(pipelineModel.Edges) == "" {
		pipelineModel.Edges = "[]"
	}
	if pipelineModel.Viewport != nil && strings.TrimSpace(*pipelineModel.Viewport) == "" {
		pipelineModel.Viewport = nil
	}

	if err := ValidateDefinition(pipelineModel.Nodes, pipelineModel.Edges); err != nil {
		return err
	}

	return nil
}

func ValidateDefinition(nodesJSON string, edgesJSON string) error {
	if strings.TrimSpace(nodesJSON) == "" {
		nodesJSON = "[]"
	}
	if strings.TrimSpace(edgesJSON) == "" {
		edgesJSON = "[]"
	}

	var nodes []flowNode
	if err := json.Unmarshal([]byte(nodesJSON), &nodes); err != nil {
		return fmt.Errorf("invalid nodes JSON: %w", err)
	}

	var edges []flowEdge
	if err := json.Unmarshal([]byte(edgesJSON), &edges); err != nil {
		return fmt.Errorf("invalid edges JSON: %w", err)
	}

	nodeTypes := make(map[string]string, len(nodes))
	returnCount := 0
	for _, flowNode := range nodes {
		nodeID := strings.TrimSpace(flowNode.ID)
		if nodeID == "" {
			return fmt.Errorf("all nodes must have an id")
		}
		if _, exists := nodeTypes[nodeID]; exists {
			return fmt.Errorf("duplicate node id %q", nodeID)
		}

		nodeType := resolveNodeType(flowNode)
		nodeTypes[nodeID] = nodeType

		if nodeType == "logic:return" {
			returnCount++
		}
	}
	if returnCount > 1 {
		return fmt.Errorf("only one Return node is allowed per pipeline")
	}

	for _, edge := range edges {
		if err := validateFlowEdge(edge, nodeTypes); err != nil {
			return err
		}
	}

	return nil
}

func BuildListOutput(pipelines []models.Pipeline, includeDefinition bool) (map[string]any, error) {
	items := make([]map[string]any, 0, len(pipelines))
	for _, pipelineModel := range pipelines {
		item, err := BuildPipelineOutput(pipelineModel, includeDefinition)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return map[string]any{
		"count":     len(items),
		"pipelines": items,
	}, nil
}

func BuildPipelineOutput(pipelineModel models.Pipeline, includeDefinition bool) (map[string]any, error) {
	result := map[string]any{
		"id":          pipelineModel.ID,
		"name":        pipelineModel.Name,
		"description": pipelineModel.Description,
		"status":      pipelineModel.Status,
		"created_at":  pipelineModel.CreatedAt,
		"updated_at":  pipelineModel.UpdatedAt,
	}

	if !includeDefinition {
		return result, nil
	}

	nodes, err := decodeJSONValue(pipelineModel.Nodes, []any{})
	if err != nil {
		return nil, fmt.Errorf("decode pipeline %s nodes: %w", pipelineModel.ID, err)
	}
	edges, err := decodeJSONValue(pipelineModel.Edges, []any{})
	if err != nil {
		return nil, fmt.Errorf("decode pipeline %s edges: %w", pipelineModel.ID, err)
	}

	result["nodes"] = nodes
	result["edges"] = edges

	if pipelineModel.Viewport != nil && strings.TrimSpace(*pipelineModel.Viewport) != "" {
		viewport, err := decodeJSONValue(*pipelineModel.Viewport, map[string]any{})
		if err != nil {
			return nil, fmt.Errorf("decode pipeline %s viewport: %w", pipelineModel.ID, err)
		}
		result["viewport"] = viewport
	}

	return result, nil
}

func CanonicalizeJSON(raw json.RawMessage, fallback string, fieldName string) (string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return fallback, nil
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("parse %s: %w", fieldName, err)
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode %s: %w", fieldName, err)
	}

	return string(encoded), nil
}

func CanonicalizeJSONPointer(raw json.RawMessage, fieldName string) (*string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	value, err := CanonicalizeJSON(raw, "", fieldName)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func resolveNodeType(flowNode flowNode) string {
	if nodeType := strings.TrimSpace(flowNode.Type); nodeType != "" {
		return nodeType
	}

	if len(flowNode.Data) == 0 {
		return ""
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(flowNode.Data, &payload); err != nil {
		return ""
	}

	rawType, ok := payload["type"]
	if !ok {
		return ""
	}

	var nodeType string
	if err := json.Unmarshal(rawType, &nodeType); err != nil {
		return ""
	}

	return strings.TrimSpace(nodeType)
}

func validateFlowEdge(edge flowEdge, nodeTypes map[string]string) error {
	edgeID := strings.TrimSpace(edge.ID)
	if edgeID == "" {
		edgeID = fmt.Sprintf("%s->%s", edge.Source, edge.Target)
	}

	sourceID := strings.TrimSpace(edge.Source)
	targetID := strings.TrimSpace(edge.Target)
	if sourceID == "" || targetID == "" {
		return fmt.Errorf("edge %q must include source and target", edgeID)
	}

	sourceType, ok := nodeTypes[sourceID]
	if !ok {
		return fmt.Errorf("edge %q references unknown source node %q", edgeID, sourceID)
	}
	targetType, ok := nodeTypes[targetID]
	if !ok {
		return fmt.Errorf("edge %q references unknown target node %q", edgeID, targetID)
	}

	if isToolEdge(edge) {
		if sourceType != "llm:agent" {
			return fmt.Errorf("edge %q uses the tool handle, but source node %q is %q instead of llm:agent", edgeID, sourceID, sourceType)
		}
		if !isToolNodeType(targetType) {
			return fmt.Errorf("edge %q uses the tool handle, but target node %q is %q instead of a tool node", edgeID, targetID, targetType)
		}
		return nil
	}

	if isToolNodeType(sourceType) {
		return fmt.Errorf("tool node %q (%s) cannot be part of the main execution chain; connect it from an LLM Agent tool handle instead", sourceID, sourceType)
	}
	if isToolNodeType(targetType) {
		return fmt.Errorf("tool node %q (%s) cannot be part of the main execution chain; connect it from an LLM Agent tool handle instead", targetID, targetType)
	}
	if sourceType == "logic:return" {
		return fmt.Errorf("Return node %q cannot have outgoing edges", sourceID)
	}
	if isTriggerNodeType(targetType) {
		return fmt.Errorf("trigger node %q (%s) cannot have incoming edges", targetID, targetType)
	}

	return nil
}

func isToolEdge(edge flowEdge) bool {
	return strings.TrimSpace(edge.SourceHandle) == "tool"
}

func isToolNodeType(nodeType string) bool {
	return strings.HasPrefix(strings.TrimSpace(nodeType), "tool:")
}

func isTriggerNodeType(nodeType string) bool {
	return strings.HasPrefix(strings.TrimSpace(nodeType), "trigger:")
}

func decodeJSONValue(raw string, fallback any) (any, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}

	if value == nil {
		return fallback, nil
	}

	return value, nil
}

func normalizeStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func isValidStatus(status string) bool {
	switch status {
	case StatusDraft, StatusActive, StatusArchived:
		return true
	default:
		return false
	}
}

func (s *Service) reload(ctx context.Context) {
	if s.reloader != nil {
		_ = s.reloader.Reload(ctx)
	}
}
