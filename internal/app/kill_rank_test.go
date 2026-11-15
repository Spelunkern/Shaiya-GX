package app

import "testing"

func TestKillRank(t *testing.T) {
	for _, c := range []struct {
		kills  uint32
		rank   int
		ranked bool
	}{
		{0, 0, false}, {1, 0, true}, {49, 0, true}, {50, 1, true},
		{999, 2, true}, {1000, 3, true}, {150000, 14, true}, {200000, 15, true},
		{999999, 29, true}, {1000000, 30, true}, {5000000, 30, true},
	} {
		rank, ranked := killRank(c.kills)
		if rank != c.rank || ranked != c.ranked {
			t.Errorf("killRank(%d) = %d,%v; want %d,%v", c.kills, rank, ranked, c.rank, c.ranked)
		}
	}
}
