// cmd/novakey/target_policy_test.go
package main

import "testing"

func TestNormalizeProcName_StripsExeAppAndPath(t *testing.T) {
	cases := map[string]string{
		`msedge.exe`:            "msedge",
		`C:\Path\To\msedge.exe`: "msedge",
		`C:\Program Files\Google\Chrome\chrome.exe`: "chrome",
		`/usr/bin/firefox`:                          "firefox",
		`Safari.app`:                                "safari",
		`/Applications/Safari.app`:                  "safari",
		`  chrome.exe  `:                            "chrome",
		``:                                          "",
	}

	for in, want := range cases {
		got := normalizeProcName(in)
		if got != want {
			t.Fatalf("normalizeProcName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestNormalizeProcList_DedupAndNormalize(t *testing.T) {
	in := []string{"msedge.exe", "MSEDGE", "  chrome  ", "chrome.exe", "/usr/bin/firefox", "firefox", "", "   "}
	out := normalizeProcList(in)

	seen := map[string]bool{}
	for _, x := range out {
		seen[x] = true
	}

	if !seen["msedge"] || !seen["chrome"] || !seen["firefox"] {
		t.Fatalf("missing expected normalized items: %#v", out)
	}
}

func TestTitleMatchesAny_SubstringCaseInsensitive(t *testing.T) {
	titleLower := "osbornepro - profile 1 - microsoft edge"
	patterns := []string{"microsoft edge", "firefox"}

	if !titleMatchesAny(titleLower, patterns) {
		t.Fatalf("expected substring match")
	}
	if titleMatchesAny(titleLower, []string{"safari"}) {
		t.Fatalf("did not expect match")
	}
}
