package importer

import (
    "context"
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type fakeProvider struct {
    products []catalog.Product
}

func (f fakeProvider) ID() string { return "dia" }
func (f fakeProvider) Search(context.Context, string, string) ([]catalog.Product, error) {
    return f.products, nil
}

type fakeSink struct {
    saved []catalog.Product
}

func (s *fakeSink) SaveProducts(_ context.Context, products []catalog.Product) error {
    s.saved = append(s.saved, products...)
    return nil
}

func TestRunPersistsProducts(t *testing.T) {
    provider := fakeProvider{products: []catalog.Product{{ID: "dia-1"}, {ID: "dia-2"}}}
    sink := &fakeSink{}

    result, err := Run(context.Background(), provider, sink, "catalog", "28001")
    if err != nil {
        t.Fatal(err)
    }
    if result.Found != 2 || result.Saved != 2 {
        t.Fatalf("unexpected result: %+v", result)
    }
    if len(sink.saved) != 2 {
        t.Fatalf("expected 2 saved products, got %d", len(sink.saved))
    }
}
