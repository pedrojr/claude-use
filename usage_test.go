package main

import (
	"testing"
	"time"
)

func TestResetText(t *testing.T) {
	setLanguage("en")
	defer setLanguage(defaultLanguage)

	// A small margin keeps the rounding stable while the test runs.
	in := func(d time.Duration) time.Time { return time.Now().Add(d + 2*time.Second) }
	cases := []struct {
		at   time.Time
		want string
	}{
		{time.Time{}, ""},
		{time.Now().Add(-time.Minute), "Resetting…"},
		{in(10 * time.Second), "Resets in 1 min"},
		{in(42 * time.Minute), "Resets in 42 min"},
		{in(4*time.Hour + 8*time.Minute), "Resets in 4 h 8 min"},
		{in(3 * 24 * time.Hour), "Resets in 3 d"},
		{in(6*24*time.Hour + 18*time.Hour), "Resets in 6 d 18 h"},
	}
	for _, c := range cases {
		if got := resetText(c.at); got != c.want {
			t.Errorf("resetText(now%+v) = %q, want %q", time.Until(c.at).Round(time.Second), got, c.want)
		}
	}
}

func TestLimitResetTime(t *testing.T) {
	var nilLimit *Limit
	if !nilLimit.ResetTime().IsZero() {
		t.Error("nil limit should have no reset time")
	}
	bad := "not a date"
	if !(&Limit{ResetsAt: &bad}).ResetTime().IsZero() {
		t.Error("invalid date should give zero time")
	}
	s := "2026-09-28T15:00:00.123456+00:00"
	want := time.Date(2026, 9, 28, 15, 0, 0, 123456000, time.UTC)
	if got := (&Limit{ResetsAt: &s}).ResetTime(); !got.Equal(want) {
		t.Errorf("ResetTime() = %v, want %v", got, want)
	}
}

// Every language must translate every key, and with the same format verbs.
func TestMessagesComplete(t *testing.T) {
	en := messages[defaultLanguage]
	for _, l := range languages {
		m, ok := messages[l.code]
		if !ok {
			t.Errorf("language %q has no messages", l.code)
			continue
		}
		for k, v := range en {
			tr, ok := m[k]
			if !ok {
				t.Errorf("%s: missing key %q", l.code, k)
			} else if verbs(tr) != verbs(v) {
				t.Errorf("%s: key %q has verbs %q, English has %q", l.code, k, verbs(tr), verbs(v))
			}
		}
		for k := range m {
			if _, ok := en[k]; !ok {
				t.Errorf("%s: key %q is not in English", l.code, k)
			}
		}
	}
}

func verbs(s string) string {
	var out []byte
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '%' {
			out = append(out, s[i+1])
			i++
		}
	}
	return string(out)
}

func TestSetLanguageFallback(t *testing.T) {
	defer setLanguage(defaultLanguage)
	setLanguage("xx")
	if languageCode() != defaultLanguage {
		t.Errorf("unknown language should fall back to %q, got %q", defaultLanguage, languageCode())
	}
}
