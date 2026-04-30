package agentmeta

import (
	"strings"
	"testing"
)

func TestParseTrailerExtractsAllFields(t *testing.T) {
	message := "Apply: ENG-1: Apply feature\n\nProduced-By: codex\nProduced-Model: gpt-5-codex\nProduced-Stage: apply\n"
	got, err := ParseTrailer(message)
	if err != nil {
		t.Fatalf("ParseTrailer returned error: %v", err)
	}
	want := Producer{By: "codex", Model: "gpt-5-codex", Stage: StageApply}
	if got != want {
		t.Fatalf("ParseTrailer = %+v, want %+v", got, want)
	}
}

func TestParseTrailerIsCaseInsensitiveOnKey(t *testing.T) {
	message := "subject\n\nproduced-by: codex\nPRODUCED-MODEL: gpt-5\nProduced-Stage: proposal\n"
	got, err := ParseTrailer(message)
	if err != nil {
		t.Fatalf("ParseTrailer returned error: %v", err)
	}
	if got.By != "codex" || got.Model != "gpt-5" || got.Stage != StageProposal {
		t.Fatalf("ParseTrailer = %+v", got)
	}
}

func TestParseTrailerReturnsNotFoundWhenAbsent(t *testing.T) {
	message := "subject line only"
	_, err := ParseTrailer(message)
	if err != ErrTrailerNotFound {
		t.Fatalf("err = %v, want ErrTrailerNotFound", err)
	}
}

func TestParseTrailerRejectsUnknownStage(t *testing.T) {
	message := "subject\n\nProduced-By: codex\nProduced-Model: gpt-5\nProduced-Stage: deploy\n"
	_, err := ParseTrailer(message)
	if err == nil {
		t.Fatal("ParseTrailer returned nil error for unknown stage")
	}
}

func TestParseTrailerIgnoresUnrelatedTrailers(t *testing.T) {
	message := "subject\n\nSigned-off-by: Alice <alice@example.com>\nProduced-By: codex\nProduced-Model: gpt-5-codex\nProduced-Stage: archive\nReviewed-by: Bob"
	got, err := ParseTrailer(message)
	if err != nil {
		t.Fatalf("ParseTrailer returned error: %v", err)
	}
	if got.Stage != StageArchive {
		t.Fatalf("Stage = %s, want archive", got.Stage)
	}
}

func TestAppendTrailerAddsAllThreeFields(t *testing.T) {
	got := AppendTrailer("Apply feature", Producer{
		By: "codex", Model: "gpt-5-codex", Stage: StageApply,
	})
	roundTrip, err := ParseTrailer(got)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v\nmessage:\n%s", err, got)
	}
	want := Producer{By: "codex", Model: "gpt-5-codex", Stage: StageApply}
	if roundTrip != want {
		t.Fatalf("round-trip = %+v, want %+v", roundTrip, want)
	}
}

func TestAppendTrailerSeparatesSubjectAndTrailerBlockWithBlankLine(t *testing.T) {
	got := AppendTrailer("Subject line", Producer{By: "codex", Model: "gpt-5", Stage: StageProposal})
	wantLines := []string{
		"Subject line",
		"",
		"Produced-By: codex",
		"Produced-Model: gpt-5",
		"Produced-Stage: proposal",
	}
	gotLines := splitLines(got)
	if !equalStringSlices(gotLines, wantLines) {
		t.Fatalf("AppendTrailer lines = %#v\nwant %#v", gotLines, wantLines)
	}
}

func TestAppendTrailerPreservesExistingTrailerBlock(t *testing.T) {
	original := "Subject\n\nSigned-off-by: Alice <alice@example.com>\n"
	got := AppendTrailer(original, Producer{By: "codex", Model: "gpt-5", Stage: StageProposal})
	roundTrip, err := ParseTrailer(got)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v", err)
	}
	if roundTrip.By != "codex" {
		t.Fatalf("By = %s, want codex", roundTrip.By)
	}
	if !strings.Contains(got, "Signed-off-by: Alice") {
		t.Fatalf("AppendTrailer dropped existing trailer block:\n%s", got)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimRight(s, "\n")
	return strings.Split(s, "\n")
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
