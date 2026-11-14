package app

import "testing"

func TestDetailsReflectionBudgetKeepsDefault(t *testing.T) {
	if waterReflectionDivisor(2) != 2 || waterReflectionDivisor(1) != 4 || waterReflectionDivisor(0) != 8 {
		t.Fatal("reflection budgets changed default or do not scale")
	}
}
