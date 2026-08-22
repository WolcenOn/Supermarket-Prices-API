package main

import (
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
