package access

import "testing"

func TestAllowConfigure(t *testing.T) {
	if !AllowConfigure(nil, "andy", false) {
		t.Fatal("first configuration is open to a local user")
	}
	if AllowConfigure(nil, "andy", true) {
		t.Fatal("later configuration needs a setup user")
	}
	if !AllowConfigure([]string{"andy"}, "andy", true) {
		t.Fatal("setup user")
	}
	if AllowConfigure(nil, "xgb-demo", false) {
		t.Fatal("backup account")
	}
	if !AllowConfigure(nil, "xgbackup", true) || !AllowConfigure(nil, "root", true) {
		t.Fatal("console")
	}
}
