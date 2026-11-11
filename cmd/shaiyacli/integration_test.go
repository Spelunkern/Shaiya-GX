//go:build integration

// N2: plays the whole flow against a running Rapture.
//
//	SHAIYA_TEST_USER=admin1 SHAIYA_PASSWORD=... go test -tags integration ./cmd/shaiyacli
package main

import (
	"os"
	"testing"
	"time"
)

func TestFlowAgainstRapture(t *testing.T) {
	user, pw := os.Getenv("SHAIYA_TEST_USER"), os.Getenv("SHAIYA_PASSWORD")
	if user == "" || pw == "" {
		t.Skip("SHAIYA_TEST_USER / SHAIYA_PASSWORD not set")
	}
	// No movement: the test must not move the account's characters.
	if err := run("127.0.0.1:30800", 30810, user, pw, 0, -1, 0, "", "shaiyacli integration", 0, time.Second, false); err != nil {
		t.Fatal(err)
	}
}
