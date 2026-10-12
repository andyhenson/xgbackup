package access

import "testing"

func TestAllow(t *testing.T) {
	rules := []RestoreRule{{User: "sam", Client: "xgb-demo", Prefix: "docs"}}
	if !AllowSetup([]string{"andy"}, "andy") || !AllowSetup(nil, "root") {
		t.Fatal("setup")
	}
	if AllowSetup([]string{"andy"}, "xgb-demo") {
		t.Fatal("backup key is not setup")
	}
	if !AllowRestore([]string{"andy"}, nil, "andy", "xgb-demo", "secret") {
		t.Fatal("setup restores all")
	}
	if !AllowRestore(nil, rules, "sam", "xgb-demo", "docs/a.txt") {
		t.Fatal("prefix")
	}
	if AllowRestore(nil, rules, "sam", "xgb-demo", "mail/a.txt") {
		t.Fatal("outside prefix")
	}
	if AllowRestore(nil, rules, "sam", "xgb-laptop", "docs/a.txt") {
		t.Fatal("other client")
	}
	if !AllowRestore(nil, []RestoreRule{{User: "sam", Client: "*", Prefix: ""}}, "sam", "xgb-laptop", "x") {
		t.Fatal("restore all")
	}
}
