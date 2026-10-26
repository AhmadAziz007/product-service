package service

import (
	"testing"

	"ecommerce/product-service/internal/core/domain/entity"
)

func TestCanTransitionProductStatus(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		allowed bool
	}{
		{name: "pending to approved", current: entity.Product_Need_Approval, next: entity.Product_Approved, allowed: true},
		{name: "pending to revision", current: entity.Product_Need_Approval, next: entity.Product_Need_Revision, allowed: true},
		{name: "pending to rejected", current: entity.Product_Need_Approval, next: entity.Product_Rejected, allowed: true},
		{name: "approved to pending", current: entity.Product_Approved, next: entity.Product_Need_Approval, allowed: true},
		{name: "approved to revision", current: entity.Product_Approved, next: entity.Product_Need_Revision, allowed: true},
		{name: "approved to rejected", current: entity.Product_Approved, next: entity.Product_Rejected, allowed: true},
		{name: "rejected to revision", current: entity.Product_Rejected, next: entity.Product_Need_Revision, allowed: true},
		{name: "revision to pending", current: entity.Product_Need_Revision, next: entity.Product_Need_Approval, allowed: true},
		{name: "approved to approved is a no-op", current: entity.Product_Approved, next: entity.Product_Approved, allowed: true},
		{name: "revision cannot go to approved", current: entity.Product_Need_Revision, next: entity.Product_Approved, allowed: false},
		{name: "rejected cannot go to approved", current: entity.Product_Rejected, next: entity.Product_Approved, allowed: false},
		{name: "rejected cannot skip revision", current: entity.Product_Rejected, next: entity.Product_Approved, allowed: false},
		{name: "revision cannot skip pending", current: entity.Product_Need_Revision, next: entity.Product_Approved, allowed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if allowed := canTransitionProductStatus(test.current, test.next); allowed != test.allowed {
				t.Errorf("canTransitionProductStatus(%q, %q) = %t, want %t", test.current, test.next, allowed, test.allowed)
			}
		})
	}
}
