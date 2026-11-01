package veriff

import (
	"encoding/json"
	"errors"
	"strings"
)

var ErrInvalidWebhook = errors.New("veriff: invalid webhook payload")

type DecisionWebhook struct {
	Verification struct {
		ID         string `json:"id"`
		Code       int    `json:"code"`
		Status     string `json:"status"`
		VendorData string `json:"vendorData"`
		Person     struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			DateOfBirth string `json:"dateOfBirth"`
			Country   string `json:"country"`
		} `json:" person,omitempty"`
	} `json:"verification"`
}

// ParseDecisionWebhook parses a signed webhook body. Call only after signature verification.
// It intentionally does not expose document images or persist raw identity attributes.
func ParseDecisionWebhook(raw []byte) (DecisionWebhook, error) {
	var event DecisionWebhook
	if len(raw) == 0 || json.Unmarshal(raw, &event) != nil {
		return DecisionWebhook{}, ErrInvalidWebhook
	}
	event.Verification.Status = strings.ToLower(strings.TrimSpace(event.Verification.Status))
	if event.Verification.ID == "" || event.Verification.Status == "" {
		return DecisionWebhook{}, ErrInvalidWebhook
	}
	return event, nil
}
