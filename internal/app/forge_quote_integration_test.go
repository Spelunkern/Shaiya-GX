//go:build integration

package app

import (
	"testing"
	"time"
)

func awaitForgeConfirmation(t *testing.T, a *App) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !a.gemConfirm.Visible {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("authoritative forge confirmation timed out")
		}
		time.Sleep(time.Millisecond)
	}
}
