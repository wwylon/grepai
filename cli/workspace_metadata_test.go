package cli

import "testing"

func TestParseMetadataFlags(t *testing.T) {
	got, err := parseMetadataFlags([]string{"upstream=frontend", "ref=feature"})
	if err != nil {
		t.Fatal(err)
	}
	if got["upstream"] != "frontend" || got["ref"] != "feature" {
		t.Fatalf("%v", got)
	}
	if _, err := parseMetadataFlags([]string{"bad"}); err == nil {
		t.Fatal("expected error")
	}
}
