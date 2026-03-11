package app

import "testing"

func TestThrottledPRFetchBatchesCorrectly(t *testing.T) {
	repos := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	maxConcurrent := 3

	batches := makeBatches(repos, maxConcurrent)
	if len(batches) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(batches))
	}
	if len(batches[0]) != 3 {
		t.Errorf("first batch should have 3, got %d", len(batches[0]))
	}
	if len(batches[2]) != 2 {
		t.Errorf("last batch should have 2, got %d", len(batches[2]))
	}
}
