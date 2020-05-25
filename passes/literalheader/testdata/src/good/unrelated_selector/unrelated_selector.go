package unrelated_selector

import "os"

func StdoutSelection() {
	_, w, _ := os.Pipe()
	os.Stdout = w // ok
}