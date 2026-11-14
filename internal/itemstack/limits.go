package itemstack

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
)

// ParseLimits reads the authoritative stackability exported from server item
// definitions. Legacy Item.SData uses byte counts and differs for some items.
func ParseLimits(data []byte) (map[uint32]uint16, error) {
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 || len(rows[0]) != 2 || rows[0][0] != "item_id" || rows[0][1] != "max_stack" {
		return nil, fmt.Errorf("invalid item stack header")
	}
	result := make(map[uint32]uint16, len(rows)-1)
	for i, row := range rows[1:] {
		if len(row) != 2 {
			return nil, fmt.Errorf("invalid stack row %d", i+2)
		}
		id, e := strconv.ParseUint(row[0], 10, 32)
		if e != nil {
			return nil, e
		}
		n, e := strconv.ParseUint(row[1], 10, 16)
		if e != nil {
			return nil, e
		}
		if id == 0 || (n != 1 && n != Max) || result[uint32(id)] != 0 {
			return nil, fmt.Errorf("invalid item stack row %d", i+2)
		}
		result[uint32(id)] = uint16(n)
	}
	return result, nil
}
