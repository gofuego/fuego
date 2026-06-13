package cli

import "testing"

func TestIsGoIdentifier(t *testing.T) {
	valid := []string{"adr", "devops", "myPack", "_x", "pack2", "k8s"}
	for _, s := range valid {
		if !isGoIdentifier(s) {
			t.Errorf("%q should be a valid identifier", s)
		}
	}
	// Note: "v2" is a valid identifier syntactically; a /vN module path
	// mismatch is a semantic case the --pack-symbol flag covers.
	invalid := []string{"", "2pack", "my-pack", "my.pack", "go", "func", "type"}
	for _, s := range invalid {
		if isGoIdentifier(s) {
			t.Errorf("%q should be rejected (needs --pack-symbol)", s)
		}
	}
}
