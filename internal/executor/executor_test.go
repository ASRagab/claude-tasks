package executor

import (
	"regexp"
	"strings"
	"testing"
)

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestGenerateUUIDFormat(t *testing.T) {
	id, err := generateUUID()
	if err != nil {
		t.Fatalf("generate UUID: %v", err)
	}
	if !uuidRE.MatchString(id) {
		t.Fatalf("expected UUIDv4 format, got %q", id)
	}
}

func TestGenerateUUIDUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 200)
	for i := 0; i < 200; i++ {
		id, err := generateUUID()
		if err != nil {
			t.Fatalf("generate UUID: %v", err)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate UUID generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestCappedBufferTruncatesOutputAndAppendsMarker(t *testing.T) {
	buf := newCappedBuffer(5)

	n, err := buf.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len("hello world") {
		t.Fatalf("expected write count %d, got %d", len("hello world"), n)
	}

	if got := buf.String(); got != "hello\n...[truncated]" {
		t.Fatalf("expected truncated output marker, got %q", got)
	}
}

func TestCappedBufferWithoutTruncationPreservesOutput(t *testing.T) {
	buf := newCappedBuffer(64)
	payload := "small output"

	n, err := buf.Write([]byte(payload))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len(payload) {
		t.Fatalf("expected write count %d, got %d", len(payload), n)
	}
	if got := buf.String(); got != payload {
		t.Fatalf("expected %q, got %q", payload, got)
	}
}

func TestCappedBufferZeroLimitAlwaysTruncates(t *testing.T) {
	buf := newCappedBuffer(0)

	_, err := buf.Write([]byte("abc"))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if got := buf.String(); !strings.Contains(got, "...[truncated]") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}
