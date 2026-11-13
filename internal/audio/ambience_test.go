package audio

import (
	"errors"
	"testing"
	"time"
)

func TestMapDecodeBoundedAndOffOwner(t *testing.T) {
	release := make(chan struct{})
	e := &Engine{cache: map[string]entry{}, failed: map[string]error{}, read: func(string) ([]byte, error) { <-release; return nil, errors.New("missing fixture") }}
	for _, name := range []string{"a", "b", "c"} {
		if _, ready, err := e.mapPCM(name); ready || err != nil {
			t.Fatal(ready, err)
		}
	}
	if len(e.pending) != 2 {
		t.Fatal("decoder concurrency is not bounded")
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for len(e.pending) > 0 {
		for _, file := range []string{"a", "b"} {
			if _, ready, err := e.mapPCM(file); ready {
				t.Fatal("invalid fixture decoded")
			} else if err != nil && e.failed[file] == nil {
				t.Fatal("decode failure was not cached")
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("decode did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}
