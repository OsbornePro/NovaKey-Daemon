// cmd/novakey/target_policy_test.go
package main

import "testing"

func TestNormProc_StripsExeAppAndPath(t *testing.T) {
	cases := map[string]string{
		`msedge.exe`:                 "msedge",
		`C:\Path\To\msedge.exe`:      "msedge",
		`/usr/bin/firefox`:           "firefox",
		`Safari.app`:                 "safari",
		`/Applications/Safari.app`:   "safari",
		`  chrome.exe  `:             "chrome",
	}

	for in, want := range cases {
		got := normProc(in)
		if got != want {
			t.Fatalf("normProc(%q)=%q want %q", in, got, want)
		}
	}
}

func TestMatchProc_AllowsBaseVsExe(t *testing.T) {
	allow := []string{"msedge", "firefox", "chrome"}

	if !matchProc("msedge.exe", allow) {
		t.Fatalf("expected msedge.exe to match allowlist containing msedge")
	}
	if !matchProc("C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe", allow) {
		t.Fatalf("expected chrome.exe path to match allowlist containing chrome")
	}
	if matchProc("unknown.exe", allow) {
		t.Fatalf("did not expect unknown.exe to match")
	}
}

func TestMatchProc_LiteralExeAlsoWorks(t *testing.T) {
	allow := []string{"msedge.exe"} // some users will write it this way

	if !matchProc("msedge.exe", allow) {
		t.Fatalf("expected msedge.exe to match allowlist containing msedge.exe")
	}
	if !matchProc("msedge", allow) {
		t.Fatalf("expected msedge to match allowlist containing msedge.exe (normalized)")
	}
}

func TestMatchTitle_Substring(t *testing.T) {
	titles := []string{"microsoft edge", "firefox"}
	if !matchTitle("osbornepro - profile 1 - microsoft edge", titles) {
		t.Fatalf("expected substring match")
	}
}

