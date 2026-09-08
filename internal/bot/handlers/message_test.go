package handlers

import (
	"testing"
	"time"
)

func TestSleepSegmentDelay(t *testing.T) {
	cases := []struct {
		name    string
		segment string
		prefix  string
		want    time.Duration
		ok      bool
	}{
		{"five seconds", "-sleep 5s", "-", 5 * time.Second, true},
		{"bare seconds", "-sleep 90", "-", 90 * time.Second, true},
		{"compound duration", "-sleep 1m30s", "-", 90 * time.Second, true},
		{"multiple args summed", "-sleep 2s 3s", "-", 5 * time.Second, true},
		{"case-insensitive command", "-SLEEP 5s", "-", 5 * time.Second, true},
		{"capped at max", "-sleep 1h", "-", maxChainSleep, true},
		{"no args is not sleep", "-sleep", "-", 0, false},
		{"malformed duration falls through", "-sleep nope", "-", 0, false},
		{"zero duration falls through", "-sleep 0s", "-", 0, false},
		{"not a sleep command", "-nsfw", "-", 0, false},
		{"custom prefix", "!sleep 5s", "!", 5 * time.Second, true},
		{"no prefix is not a segment", "sleep 5s", "-", 0, false},
		{"no segment", "&&", "-", 0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := sleepSegmentDelay(tc.segment, tc.prefix)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("delay = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseSleepDuration(t *testing.T) {
	cases := []struct {
		raw  string
		want time.Duration
		ok   bool
	}{
		{"5s", 5 * time.Second, true},
		{"1m30s", 90 * time.Second, true},
		{"2h", 2 * time.Hour, true},
		{"7", 7 * time.Second, true},
		{"0", 0, true},
		{"", 0, false},
		{"nope", 0, false},
		{"-5s", -5 * time.Second, true},
	}

	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := parseSleepDuration(tc.raw)
			if tc.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("expected error, got %v", got)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSleepTracker(t *testing.T) {
	key := sleepKey{guildID: "g1", userID: "u1"}

	t.Run("pop empty", func(t *testing.T) {
		tr := newSleepTracker()
		if rem := tr.pop(key); rem != 0 {
			t.Fatalf("pop on empty tracker = %v, want 0", rem)
		}
	})

	t.Run("pending sleep is returned once", func(t *testing.T) {
		tr := newSleepTracker()
		tr.set(key, 50*time.Millisecond)
		rem := tr.pop(key)
		if rem <= 0 || rem > 50*time.Millisecond {
			t.Fatalf("rem = %v, want 0 < rem <= 50ms", rem)
		}
		if again := tr.pop(key); again != 0 {
			t.Fatalf("second pop = %v, want 0 (consumed)", again)
		}
	})

	t.Run("expired sleep is not returned", func(t *testing.T) {
		tr := newSleepTracker()
		tr.set(key, 10*time.Millisecond)
		time.Sleep(30 * time.Millisecond)
		if rem := tr.pop(key); rem != 0 {
			t.Fatalf("pop after expiry = %v, want 0", rem)
		}
	})

	t.Run("per-guild isolation", func(t *testing.T) {
		tr := newSleepTracker()
		tr.set(sleepKey{guildID: "g1", userID: "u1"}, time.Minute)
		if rem := tr.pop(sleepKey{guildID: "g2", userID: "u1"}); rem != 0 {
			t.Fatalf("other guild popped = %v, want 0", rem)
		}
		if rem := tr.pop(sleepKey{guildID: "g1", userID: "u2"}); rem != 0 {
			t.Fatalf("other user popped = %v, want 0", rem)
		}
	})

	t.Run("set overwrites and prunes expired", func(t *testing.T) {
		tr := newSleepTracker()
		tr.set(sleepKey{guildID: "g1", userID: "u1"}, 10*time.Millisecond)
		tr.set(sleepKey{guildID: "g2", userID: "u2"}, 10*time.Millisecond)
		time.Sleep(30 * time.Millisecond)
		tr.set(key, 50*time.Millisecond)
		tr.mu.Lock()
		lenAfterPrune := len(tr.deadlines)
		tr.mu.Unlock()
		if lenAfterPrune != 1 {
			t.Fatalf("deadlines after prune = %d, want 1", lenAfterPrune)
		}
	})
}
