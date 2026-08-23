package main

import (
    "reflect"
    "testing"
    "time"
)

func TestBoundedLimit(t *testing.T) {
    if _, err := boundedLimit(0); err == nil {
        t.Fatal("expected zero limit to fail")
    }
    if got, err := boundedLimit(5); err != nil || got != 5 {
        t.Fatalf("expected 5, got %d err=%v", got, err)
    }
    if got, err := boundedLimit(100); err != nil || got != maxLimit {
        t.Fatalf("expected hard cap %d, got %d err=%v", maxLimit, got, err)
    }
}

func TestBoundedDelay(t *testing.T) {
    if got := boundedDelay(0); got != minDelay {
        t.Fatalf("expected minimum delay %s, got %s", minDelay, got)
    }
    if got := boundedDelay(3 * time.Second); got != 3*time.Second {
        t.Fatalf("expected explicit safe delay, got %s", got)
    }
}

func TestParseExternalIDs(t *testing.T) {
    got, err := parseExternalIDs("151, 21415,151")
    if err != nil {
        t.Fatal(err)
    }
    want := []string{"151", "21415"}
    if !reflect.DeepEqual(got, want) {
        t.Fatalf("expected %#v, got %#v", want, got)
    }

    if _, err := parseExternalIDs("151,,21415"); err == nil {
        t.Fatal("expected empty SKU to fail")
    }
    if _, err := parseExternalIDs("151,abc"); err == nil {
        t.Fatal("expected non-numeric DIA SKU to fail")
    }
}

func TestValidatePersistenceScope(t *testing.T) {
    if err := validatePersistenceScope(false, nil, 5); err != nil {
        t.Fatalf("preview without explicit SKUs should remain valid: %v", err)
    }
    if err := validatePersistenceScope(true, nil, 5); err == nil {
        t.Fatal("persist without explicit reviewed SKUs must fail")
    }
    if err := validatePersistenceScope(true, []string{"151", "21415"}, 2); err != nil {
        t.Fatalf("explicit reviewed persist scope should be valid: %v", err)
    }
    if err := validatePersistenceScope(true, []string{"151", "21415"}, 1); err == nil {
        t.Fatal("persist scope larger than limit must fail")
    }
}

func TestSameExternalID(t *testing.T) {
    if !sameExternalID(" 303311 ", "303311") {
        t.Fatal("expected matching SKU")
    }
    if sameExternalID("303311", "28809") {
        t.Fatal("must reject mismatched SKU")
    }
    if sameExternalID("", "303311") {
        t.Fatal("must reject empty stored SKU")
    }
}
