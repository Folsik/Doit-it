package ledger

import (
	"context"
	"testing"
)

func TestPostTransferRejectsInvalidInputBeforeDatabaseAccess(t *testing.T) {
	service := NewService()
	tests := []struct {
		name       string
		transferID string
		from       string
		to         string
		currency   string
		amount     int64
	}{
		{name: "missing transfer", from: "from", to: "to", currency: "EUR", amount: 100},
		{name: "missing source", transferID: "transfer", to: "to", currency: "EUR", amount: 100},
		{name: "missing destination", transferID: "transfer", from: "from", currency: "EUR", amount: 100},
		{name: "same account", transferID: "transfer", from: "same", to: "same", currency: "EUR", amount: 100},
		{name: "missing currency", transferID: "transfer", from: "from", to: "to", amount: 100},
		{name: "zero amount", transferID: "transfer", from: "from", to: "to", currency: "EUR"},
		{name: "negative amount", transferID: "transfer", from: "from", to: "to", currency: "EUR", amount: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.PostTransfer(context.Background(), nil, tt.transferID, tt.from, tt.to, tt.currency, tt.amount)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNewService(t *testing.T) {
	if NewService() == nil {
		t.Fatal("NewService() returned nil")
	}
}
