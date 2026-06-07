package agent

import (
	"fmt"
	"time"

	"aide/cli/internal/store"
)

type StoreDiff struct {
	NewItems      []store.Item
	ResolvedItems []store.Item
}

func ComputeDiff(s *store.Store, since time.Time) (StoreDiff, error) {
	newItems, err := s.Items.RecentlyDiscovered("", since)
	if err != nil {
		return StoreDiff{}, fmt.Errorf("querying new items: %w", err)
	}

	resolvedItems, err := s.Items.RecentlyResolved("", since)
	if err != nil {
		return StoreDiff{}, fmt.Errorf("querying resolved items: %w", err)
	}

	return StoreDiff{NewItems: newItems, ResolvedItems: resolvedItems}, nil
}
