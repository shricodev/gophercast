// Package testutil is a collection of helper functions for tests
package testutil

import (
	"strings"
	"testing"
)

func AssertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func AssertErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got: nil")
	}
}

func AssertLen(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("expected length: %d, got: %d", want, got)
	}
}

func AssertNotEmpty(t *testing.T, s string) {
	t.Helper()
	if s == "" {
		t.Fatal("expected non-empty string, got: \"\"")
	}
}

func AssertEmpty(t *testing.T, s string) {
	t.Helper()
	if s != "" {
		t.Fatalf("expected empty string, got: %s", s)
	}
}

func AssertEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("expected: %s, got: %s", want, got)
	}
}

func AssertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected: %s, got: %s", want, got)
	}
}

func AssertNotNil(t *testing.T, got any) {
	t.Helper()
	if got == nil {
		t.Fatal("expected non-nil value, got: nil")
	}
}
