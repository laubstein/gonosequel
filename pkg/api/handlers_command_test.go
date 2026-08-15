package api

import (
	"reflect"
	"testing"
)

func TestTokenizeCommand(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "GET foo", []string{"GET", "foo"}},
		{"extra whitespace", "  SET   foo   bar  ", []string{"SET", "foo", "bar"}},
		{"double quoted with space", `SET key "hello world"`, []string{"SET", "key", "hello world"}},
		{"single quoted with space", `SET key 'hello world'`, []string{"SET", "key", "hello world"}},
		{"empty quoted string", `SET key ""`, []string{"SET", "key", ""}},
		{"escaped char in double quotes", `SET key "a\"b"`, []string{"SET", "key", `a"b`}},
		{"empty line", "", nil},
		{"only whitespace", "   ", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tokenizeCommand(tc.input)
			if err != nil {
				t.Fatalf("tokenizeCommand(%q): %v", tc.input, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("tokenizeCommand(%q) = %#v, want %#v", tc.input, got, tc.want)
			}
		})
	}
}

func TestTokenizeCommandUnterminatedQuote(t *testing.T) {
	if _, err := tokenizeCommand(`SET key "unterminated`); err == nil {
		t.Error("expected an error for an unterminated quote, got nil")
	}
}
