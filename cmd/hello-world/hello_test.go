package main

import "testing"

func TestHelloWorld(t *testing.T) {
	t.Run("should say hello world", func(t *testing.T) {
		got := HelloWorld()
		want := "Hello, World!"
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	})
}
