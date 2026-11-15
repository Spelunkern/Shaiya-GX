package app

// playerPlate carries the optional parts of a player's name plate.
type playerPlate struct {
	gm    bool
	kills uint32
}

// Addresses here and in the name plate code refer to an EP5.4 Game.exe
// build of 4,000,768 bytes, not the reference build cited elsewhere in GX; the routines are identified by their behaviour.
//
// killRankKills is Game.exe's kill rank table (0x7aee08): the minimum PvP
// kills of ranks 1-31.
var killRankKills = [31]uint32{
	1, 50, 300, 1000, 5000, 10000, 20000, 30000, 40000, 50000,
	70000, 90000, 110000, 130000, 150000, 200000, 250000, 300000, 350000, 400000,
	450000, 500000, 550000, 600000, 650000, 700000, 750000, 800000, 850000, 900000,
	1000000,
}

// killRank returns the zero-based rank icon index for a kill count. Native
// 0x453a80 draws no icon without kills; 0x44ebe0 picks the highest rank reached.
func killRank(kills uint32) (int, bool) {
	for i := len(killRankKills) - 1; i >= 0; i-- {
		if kills >= killRankKills[i] {
			return i, true
		}
	}
	return 0, false
}
