package api

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// The withheld city these rows stand in for. Its length is what the mask must
// never echo, so the assertions below derive from it rather than from a
// literal width.
const withheldValue = "Copenhagen"

func veiledRow(veilID string, context string) AnalyticsRow {
	key := withheldValue
	return AnalyticsRow{
		// A veiled row arrives with its name already gone. The label is set here
		// anyway, so a renderer that reaches past Disclosure for something to
		// print fails this test instead of leaking in production.
		Label:  &key,
		Clicks: 15295,
		// UniqueClicks stays nil: the wire omits the field on veiled rows.
		Disclosure: "veiled",
		VeilID:     veilID,
		Context:    context,
	}
}

func TestVeiledRowNeverRendersItsValue(t *testing.T) {
	row := veiledRow("72f751f475c48147", "DK")
	got := row.Display("city")
	if strings.Contains(got, withheldValue) {
		t.Fatalf("veiled row rendered its withheld value: %q", got)
	}
	if !strings.Contains(got, "DK") {
		t.Fatalf("veiled row dropped the coarser place the API disclosed: %q", got)
	}
}

// The run must not encode how long the hidden value is.
func TestMaskLengthIsIndependentOfTheHiddenValue(t *testing.T) {
	dots := maskDots("city", "72f751f475c48147")
	if utf8.RuneCountInString(dots) == utf8.RuneCountInString(withheldValue) {
		t.Fatalf("mask is exactly as long as the value it hides (%q)", withheldValue)
	}
}

// Same value, same run: a row has to stay recognizable across reads while it
// waits for its crowd.
func TestMaskIsStableForOneValue(t *testing.T) {
	first := maskDots("city", "72f751f475c48147")
	second := maskDots("city", "72f751f475c48147")
	if first != second {
		t.Fatalf("mask changed between reads: %q then %q", first, second)
	}
}

// A dimension whose values vary in length gets runs that vary too, or one
// shared run length would read as one shared value.
func TestMaskVariesAcrossValuesOnAVariableLengthDimension(t *testing.T) {
	seen := map[string]bool{}
	for _, id := range []string{
		"72f751f475c48147", "b074c8ac3f4e83c4", "8a4eb32e9e3d7ec7",
		"58194eaca29da694", "97fde6d49145cb00", "e3a35e1c218ed7d9",
		"1bfaa404a6aabdf6", "aac6552fa6f1c9b2",
	} {
		seen[maskDots("city", id)] = true
	}
	if len(seen) < 2 {
		t.Fatalf("every city mask came out the same length: %v", seen)
	}
}

// Country codes are the same length everywhere, so that length is common
// knowledge and the run states it rather than pretending to vary.
func TestMaskIsUniformWhereTheValuesAre(t *testing.T) {
	first := maskDots("country", "fa69bc637458733b")
	second := maskDots("country", "33951a3825b478cc")
	if first != second {
		t.Fatalf("country masks differ (%q, %q) on a fixed-width dimension", first, second)
	}
	if utf8.RuneCountInString(first) != len("DK") {
		t.Fatalf("country mask is %d dots; an ISO country code is %d characters",
			utf8.RuneCountInString(first), len("DK"))
	}
}
