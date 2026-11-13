package text

import "testing"

func TestNativeTemplateExpansionIsOnePass(t *testing.T) {
	m := SysMsg{1: "[<p>] <b>; <v>/<v2> <hp>; <unknown>"}
	got := m.Format(1, map[string]string{"p": "Alice", "b": "literal <p>", "v": "12", "v2": "30", "hp": "MP"})
	if got != "[Alice] literal <p>; 12/30 MP; <unknown>" {
		t.Fatal(got)
	}
	if m.Format(2, nil) != "" {
		t.Fatal("missing message invented text")
	}
}
