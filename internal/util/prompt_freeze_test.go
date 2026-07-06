package util

import (
	"os"
	"testing"
)

func TestPromptNonBlockingWhenStdoutPiped(t *testing.T) {
	realOut := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = realOut
		w.Close()
		r.Close()
	}()

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := r.Read(buf); err != nil {
				break
			}
		}
		close(done)
	}()

	got := SelectOne("pick", []SelectOption{
		{Value: "a", Label: "A"},
		{Value: "b", Label: "B", Selected: true},
	})
	if got != "b" {
		t.Fatalf("SelectOne = %q, want %q", got, "b")
	}

	ms := MultiSelect("pick", []MultiSelectOption{
		{Value: "x", Label: "X", Selected: true},
		{Value: "y", Label: "Y"},
	})
	if len(ms) != 1 || ms[0] != "x" {
		t.Fatalf("MultiSelect = %v, want [x]", ms)
	}

	if !Confirm("ok?", true) {
		t.Fatal("Confirm(true) = false, want true")
	}
	if Confirm("ok?", false) {
		t.Fatal("Confirm(false) = true, want false")
	}

	os.Stdout = realOut
	w.Close()
	<-done
}