package protocol

import "testing"

func TestParseRoundTrip(t *testing.T) {
	in := Message{Type: TypeStdout, ExecID: "ex_1", Data: "hi\n"}
	b, err := in.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	out, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != TypeStdout || out.ExecID != "ex_1" || out.Data != "hi\n" {
		t.Fatalf("got %+v", out)
	}
}

func TestSHA256HexStable(t *testing.T) {
	a := SHA256Hex("xr_grant_abc")
	b := SHA256Hex("xr_grant_abc")
	if a != b || len(a) != 64 {
		t.Fatalf("hash %s", a)
	}
}
