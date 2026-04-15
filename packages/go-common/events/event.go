package events

import "time"

type Envelope struct {
	TenantID    string         `json:"tenantId"`
	Type        string         `json:"type"`
	OccurredAt  time.Time      `json:"occurredAt"`
	Payload     map[string]any `json:"payload"`
	TraceID     string         `json:"traceId,omitempty"`
	Correlation string         `json:"correlationId,omitempty"`
}
