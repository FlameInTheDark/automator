package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/FlameInTheDark/emerald/internal/nodedefs"
)

type NodeDefinitionsHandler struct {
	service  *nodedefs.Service
	reloader nodeDefinitionsReloader
}

type nodeDefinitionsResponse struct {
	Definitions []nodedefs.Definition `json:"definitions"`
	Plugins     any                   `json:"plugins,omitempty"`
	Error       string                `json:"error,omitempty"`
}

type nodeDefinitionsReloader interface {
	Reload(ctx context.Context) error
}

func NewNodeDefinitionsHandler(service *nodedefs.Service, reloaders ...nodeDefinitionsReloader) *NodeDefinitionsHandler {
	handler := &NodeDefinitionsHandler{service: service}
	if len(reloaders) > 0 {
		handler.reloader = reloaders[0]
	}
	return handler
}

func (h *NodeDefinitionsHandler) List(c *fiber.Ctx) error {
	if h == nil || h.service == nil {
		return c.JSON(nodeDefinitionsResponse{Definitions: []nodedefs.Definition{}})
	}

	return c.JSON(h.response(""))
}

func (h *NodeDefinitionsHandler) Refresh(c *fiber.Ctx) error {
	if h == nil || h.service == nil {
		return c.JSON(nodeDefinitionsResponse{Definitions: []nodedefs.Definition{}})
	}

	var errMsg string
	if refreshErr := h.service.RefreshPlugins(c.UserContext()); refreshErr != nil {
		errMsg = refreshErr.Error()
	}
	if h.reloader != nil {
		if reloadErr := h.reloader.Reload(c.UserContext()); reloadErr != nil {
			if errMsg != "" {
				errMsg += "; "
			}
			errMsg += reloadErr.Error()
		}
	}
	if errMsg != "" {
		return c.JSON(h.response(errMsg))
	}

	return c.JSON(h.response(""))
}

func (h *NodeDefinitionsHandler) response(err string) nodeDefinitionsResponse {
	return nodeDefinitionsResponse{
		Definitions: h.service.List(),
		Plugins:     h.service.PluginStatuses(),
		Error:       err,
	}
}
