package channels

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func appendChannelIDParam(baseURL string, channelID string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return baseURL
	}

	query := parsed.Query()
	query.Set("channel_id", channelID)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func buildTelegramConnectURL(baseURL string, channelID string) string {
	return appendChannelIDParam(baseURL, channelID)
}

func marshalValueToMap(value any) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func sleepContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
