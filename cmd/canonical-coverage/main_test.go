package main

import (
	"strings"
	"testing"
)

func TestParseInputsAcceptsPackIngredients(t *testing.T) {
	items, err := parseInputs([]byte(`{
		"ingredients": [
			{"name": "Calabaza"},
			{"name": "Pechuga de pollo"}
		]
	}`), "pack.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "Calabaza" || items[0].Source != "pack.json" {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
}

func TestParseInputsAcceptsExplicitCounts(t *testing.T) {
	items, err := parseInputs([]byte(`[
		{"name":"Arroz blanco","count":12,"source":"packs"}
	]`), "ignored.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Count != 12 || items[0].Source != "packs" {
		t.Fatalf("unexpected item: %+v", items)
	}
}

func TestLoadInputsUsesStdinWhenNoFilesProvided(t *testing.T) {
	items, err := loadInputs(nil, strings.NewReader(`["Tomate","Cebolla"]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].Name != "Cebolla" {
		t.Fatalf("unexpected stdin items: %+v", items)
	}
}
