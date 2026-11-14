// Package itemtype defines the original item-family mapping shared by client and server.
package itemtype

func Real(t uint8) uint8 {
	groups := [][]uint8{{1, 45}, {2, 46}, {3, 47}, {4, 48}, {5, 49, 50}, {6, 51, 52}, {7, 53, 54}, {8, 55, 56}, {9, 57},
		{10, 58}, {11, 59}, {12, 60, 61}, {13, 62, 63}, {14, 64}, {15, 65}, {16, 31, 66, 72, 81, 87},
		{17, 32, 67, 73, 82, 88}, {18, 33, 68, 74, 83, 89}, {19, 34, 69, 75, 84, 90}, {20, 35, 70, 76, 85, 91},
		{21, 36, 71, 77, 86, 92}, {22, 37}, {23, 38}, {24, 39}, {40, 41}, {25, 44, 100}, {27, 28, 29}}
	for i, g := range groups {
		for _, v := range g {
			if v == t {
				return uint8(i + 1)
			}
		}
	}
	return 0
}
