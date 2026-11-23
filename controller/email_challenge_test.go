package controller

import (
	"testing"
	"time"
)

// A login with no pending challenge must send the email; only an unexpired one suppresses it.
func TestHasActiveChallenge(t *testing.T) {
	s := NewChallengeStore()
	if s.hasActiveChallenge("new-id") {
		t.Fatal("no challenge yet: email must be sent")
	}
	s.CreateLoginChallenge("live", "a@example.com")
	if !s.hasActiveChallenge("live") {
		t.Fatal("unexpired challenge: email must not be resent")
	}
	s.Mu.Lock()
	s.cache["old"] = VerifyLoginCache{Code: "x", CreateAt: time.Now().Add(-time.Hour), Exp: time.Minute}
	s.Mu.Unlock()
	if s.hasActiveChallenge("old") {
		t.Fatal("expired challenge: email must be sent again")
	}
}
