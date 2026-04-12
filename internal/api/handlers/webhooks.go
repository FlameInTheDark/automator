package handlers

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/FlameInTheDark/emerald/internal/pipeline"
	"github.com/FlameInTheDark/emerald/internal/triggers"
)

type webhookDispatcher interface {
	DispatchWebhook(ctx context.Context, request triggers.WebhookRequest) (*pipeline.ExecutionRunResult, error)
}

type WebhookHandler struct {
	dispatcher webhookDispatcher
}

func NewWebhookHandler(dispatcher webhookDispatcher) *WebhookHandler {
	return &WebhookHandler{dispatcher: dispatcher}
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	if h == nil || h.dispatcher == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "webhook dispatcher is not configured",
		})
	}

	result, err := h.dispatcher.DispatchWebhook(c.Context(), triggers.WebhookRequest{
		Method:      c.Method(),
		Path:        c.Path(),
		Token:       extractWebhookToken(c),
		ContentType: c.Get(fiber.HeaderContentType),
		Headers:     collectRequestHeaders(c),
		Query:       collectRequestQuery(c),
		Body:        append([]byte(nil), c.Body()...),
		RemoteIP:    c.IP(),
		UserAgent:   c.Get(fiber.HeaderUserAgent),
	})
	if err != nil {
		switch err {
		case triggers.ErrWebhookNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		case triggers.ErrWebhookUnauthorized:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	response := fiber.Map{
		"execution_id": result.ExecutionID,
		"status":       result.Status,
		"duration":     result.Duration.String(),
		"nodes_run":    result.NodesRun,
	}
	if result.ErrorMessage != "" {
		response["error"] = result.ErrorMessage
	}
	if result.Returned {
		response["returned"] = true
		response["return_value"] = pipeline.SanitizeExecutionValue(result.ReturnValue)
	}

	return c.JSON(response)
}

func collectRequestHeaders(c *fiber.Ctx) map[string][]string {
	headers := make(map[string][]string)
	if c == nil {
		return headers
	}

	c.Context().Request.Header.VisitAll(func(key []byte, value []byte) {
		headers[string(key)] = append(headers[string(key)], string(value))
	})

	return headers
}

func collectRequestQuery(c *fiber.Ctx) map[string][]string {
	query := make(map[string][]string)
	if c == nil {
		return query
	}

	c.Context().QueryArgs().VisitAll(func(key []byte, value []byte) {
		query[string(key)] = append(query[string(key)], string(value))
	})

	return query
}

func extractWebhookToken(c *fiber.Ctx) string {
	if c == nil {
		return ""
	}

	if token := strings.TrimSpace(c.Get("X-Emerald-Webhook-Token")); token != "" {
		return token
	}

	if authHeader := strings.TrimSpace(c.Get(fiber.HeaderAuthorization)); strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	return strings.TrimSpace(c.Query("token"))
}
