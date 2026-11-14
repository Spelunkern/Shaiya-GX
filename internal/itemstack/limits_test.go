package itemstack

import "testing"

func TestLimitsPreserveStackability(t *testing.T) {
	values, err := ParseLimits([]byte("item_id,max_stack\n25001,255\n1001,1\n"))
	if err != nil || values[25001] != 255 || values[1001] != 1 {
		t.Fatal(values, err)
	}
	for _, s := range []string{"item_id,max_stack\n25001,1000\n", "item_id,max_stack\n25001,255\n25001,1\n", "item_id,max_stack\n25001,254\n"} {
		if _, err := ParseLimits([]byte(s)); err == nil {
			t.Fatal("invalid limits accepted")
		}
	}
	if Limit(10) != 255 || Limit(255) != 255 || Limit(1) != 1 || Limit(0) != 1 {
		t.Fatal("stackability changed")
	}
}
