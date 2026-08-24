package main

import (
	"context"
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/importer"
)

type testSink struct{}

func (s *testSink) SaveProducts(context.Context, []catalog.Product) error {
	return nil
}

func TestPersistentImportSinkCanDisableMatching(t *testing.T) {
	delegate := &testSink{}
	got := persistentImportSink(nil, delegate, false)
	if got != importer.Sink(delegate) {
		t.Fatalf("matching disabled: got %T, want original delegate", got)
	}
}

func TestPersistentImportSinkKeepsExistingMatchingDefault(t *testing.T) {
	delegate := &testSink{}
	got := persistentImportSink(nil, delegate, true)
	matching, ok := got.(*matchingSink)
	if !ok {
		t.Fatalf("matching enabled: got %T, want *matchingSink", got)
	}
	if matching.delegate != importer.Sink(delegate) {
		t.Fatalf("matching enabled: delegate was not preserved")
	}
}
