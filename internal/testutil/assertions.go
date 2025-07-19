package testutil

import (
	"strings"
	"testing"
)

// AssertNoErr fails the test if an error is not nil.
func AssertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// AssertErr fails the test if an error is nil.
func AssertErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got: nil")
	}
}

// AssertLen fails the test if the length of a slice is not equal to the
// expected length.
func AssertLen(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("expected length: %d, got: %d", want, got)
	}
}

// AssertNotEmpty fails the test if a string is empty.
func AssertNotEmpty(t *testing.T, s string) {
	t.Helper()
	if s == "" {
		t.Fatal("expected non-empty string, got: \"\"")
	}
}

// AssertEmpty fails the test if a string is not empty.
func AssertEmpty(t *testing.T, s string) {
	t.Helper()
	if s != "" {
		t.Fatalf("expected empty string, got: %s", s)
	}
}

// AssertEqual fails the test if two strings are not equal.
func AssertEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("expected: %s, got: %s", want, got)
	}
}

// AssertContains fails the test if a string does not contain a substring.
func AssertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected: %s, got: %s", want, got)
	}
}

// AssertNotNil fails the test if a value is nil.
func AssertNotNil(t *testing.T, got any) {
	t.Helper()
	if got == nil {
		t.Fatal("expected non-nil value, got: nil")
	}
}
