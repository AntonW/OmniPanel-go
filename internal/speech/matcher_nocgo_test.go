//go:build !cgo

package speech

import "testing"

func TestMatcherHelpers(t *testing.T) {
	ok, params := matchPhrase("Open Map", " open map ")
	if !ok || params != nil {
		t.Fatalf("expected exact match without params, got ok=%v params=%v", ok, params)
	}

	ok, params = matchPhrase("set {value} to {target}", "Set 50 to throttle")
	if !ok || params["value"] != "50" || params["target"] != "throttle" {
		t.Fatalf("unexpected param match: ok=%v params=%v", ok, params)
	}

	if buildRegexPattern("set {value} to {target}") != "set (\\S+) to (\\S+)" {
		t.Fatal("unexpected regex pattern")
	}

	commands := []SpeechCommand{{Phrase: "open map"}, {Phrase: "toggle gear", Aliases: []string{"gear"}}}
	ok, cmd := fuzzyMatchCommand("oppen map", commands)
	if !ok || cmd.Phrase != "open map" {
		t.Fatalf("unexpected fuzzy match: ok=%v cmd=%+v", ok, cmd)
	}

	if similarity("", "") != 1.0 {
		t.Fatal("expected similarity for empty strings to be 1")
	}
	if levenshtein("kitten", "sitting") != 3 {
		t.Fatal("unexpected levenshtein distance")
	}
	if min3(3, 1, 2) != 1 {
		t.Fatal("unexpected min3")
	}
	if !matchAllowlist("set * to *", "set throttle to 50") {
		t.Fatal("allowlist wildcard should match")
	}
	if matchAllowlist("launch", "land") {
		t.Fatal("exact allowlist should not match")
	}
}

