package entity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCartItemUnmarshalJSONSupportsLegacyAndRFC3339Timestamps(t *testing.T) {
	tests := []struct {
		name      string
		createdAt string
		updatedAt string
	}{
		{
			name:      "legacy timestamps",
			createdAt: "2026-10-03 22:29:48",
			updatedAt: "2026-10-03 22:29:48",
		},
		{
			name:      "RFC3339 timestamps",
			createdAt: "2026-10-03T22:29:48+07:00",
			updatedAt: "2026-10-03T22:29:48+07:00",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(map[string]interface{}{
				"product_id": 3,
				"quantity":   2,
				"created_at": test.createdAt,
				"updated_at": test.updatedAt,
			})
			if err != nil {
				t.Fatal(err)
			}

			var item CartItem
			if err := json.Unmarshal(data, &item); err != nil {
				t.Fatalf("unmarshal cart item: %v", err)
			}

			if item.ProductID != 3 || item.Quantity != 2 {
				t.Fatalf("unexpected cart item: %+v", item)
			}
			if item.CreatedAt.IsZero() || item.UpdatedAt.IsZero() {
				t.Fatal("expected timestamps to be parsed")
			}
			if !item.CreatedAt.Equal(item.UpdatedAt) {
				t.Fatal("expected timestamps to match")
			}
			if item.CreatedAt.Year() != 2026 || item.CreatedAt.Month() != time.October {
				t.Fatalf("unexpected timestamp: %s", item.CreatedAt)
			}
		})
	}
}
