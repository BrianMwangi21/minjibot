package setup

import (
	"strings"
	"testing"
)

func TestGuardReasons_FreshServer_Passes(t *testing.T) {
	if reasons := guardReasons(3, 0, 2); len(reasons) != 0 {
		t.Fatalf("fresh server flagged: %v", reasons)
	}
}

func TestGuardReasons_ManyChannelsOnly_Passes(t *testing.T) {
	// A lot of channels by itself is not a sign the server is live; the owner
	// may still be shaping it.
	if reasons := guardReasons(30, 0, 1); len(reasons) != 0 {
		t.Fatalf("channel-only server flagged: %v", reasons)
	}
}

func TestGuardReasons_ChannelsPlusBots_Blocked(t *testing.T) {
	reasons := guardReasons(15, 2, 3)
	if len(reasons) == 0 {
		t.Fatal("expected channels+other bots to be flagged")
	}
	if !strings.Contains(reasons.Error(), "15 channels") {
		t.Fatalf("message should mention channel count, got: %v", reasons.Error())
	}
	if !strings.Contains(reasons.Error(), "2 other bot") {
		t.Fatalf("message should mention bot count, got: %v", reasons.Error())
	}
}

func TestGuardReasons_ManyMembers_Blocked(t *testing.T) {
	reasons := guardReasons(4, 0, 21)
	if len(reasons) == 0 {
		t.Fatal("expected many members to be flagged")
	}
	if !strings.Contains(reasons.Error(), "21 members") {
		t.Fatalf("message should mention member count, got: %v", reasons.Error())
	}
}

func TestGuardReasons_ExactlyTwentyMembers_Passes(t *testing.T) {
	if reasons := guardReasons(5, 0, 20); len(reasons) != 0 {
		t.Fatalf("exactly 20 members flagged: %v", reasons)
	}
}

func TestErrors_Empty_IsNil(t *testing.T) {
	if err := (Reasons{}).err(); err != nil {
		t.Fatalf("empty reasons should be a nil error, got %v", err)
	}
}

func TestErrors_NonEmpty_IsFormatted(t *testing.T) {
	err := (Reasons{"a", "b"}).err()
	if err == nil {
		t.Fatal("expected an error")
	}
	want := "server looks already set up: a; b"
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}
