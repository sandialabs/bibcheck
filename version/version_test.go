package version

import "testing"

func TestGitRefNameFallback(t *testing.T) {
	restore := setVersionTestValues("", "", "")
	defer restore()

	if got, want := GitRefName(), "[git ref not provided]"; got != want {
		t.Fatalf("GitRefName() = %q, want %q", got, want)
	}
}

func TestStringFormatsRefAndShortSha(t *testing.T) {
	restore := setVersionTestValues("main", "abcdef1234567890", "")
	defer restore()

	if got, want := String(), "main (abcdef1)"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStringUsesShortShaAsIs(t *testing.T) {
	restore := setVersionTestValues("feature", "abc123", "")
	defer restore()

	if got, want := String(), "feature (abc123)"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStringUsesMissingShaFallback(t *testing.T) {
	restore := setVersionTestValues("main", "", "")
	defer restore()

	if got, want := String(), "main ([git SHA not provided])"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func setVersionTestValues(refName, sha, date string) func() {
	oldRefName := gitRefName
	oldSha := gitSha
	oldDate := buildDate
	gitRefName = refName
	gitSha = sha
	buildDate = date
	return func() {
		gitRefName = oldRefName
		gitSha = oldSha
		buildDate = oldDate
	}
}
