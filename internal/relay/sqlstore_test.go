package relay

import (
	"testing"
	"time"
)

func TestSQLStoreDeviceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenSQLStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.SaveDevice("dev_a", "sec", "gr", true)
	sec, gr, inb, ok := s.LoadDevice("dev_a")
	if !ok || sec != "sec" || gr != "gr" || !inb {
		t.Fatalf("load %+v %v %v %v", sec, gr, inb, ok)
	}
	s.DeleteDevice("dev_a")
	if _, _, _, ok := s.LoadDevice("dev_a"); ok {
		t.Fatal("deleted row still present")
	}
}

func TestSQLStoreAuditOmitsFullCommand(t *testing.T) {
	s, err := OpenSQLStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	s.AppendAudit(AuditEntry{
		DeviceID: "dev_a", Op: "exec", ExecID: "ex_1",
		Decision: "accept", ArgsPreview: "whoami", ArgsDigest: "abc",
	})
	time.Sleep(20 * time.Millisecond)
	var n int
	if err := s.db.QueryRow(`SELECT count(*) FROM audit WHERE args_preview=?`, "whoami").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit rows %d", n)
	}
}

func TestAuditPreviewWriteIsPathOnly(t *testing.T) {
	payload := []byte(`{"path":"a.txt","content":"SECRET-BODY-SHOULD-NOT-PREVIEW"}`)
	prev, dig := auditPreview("write", payload)
	if prev != "a.txt" {
		t.Fatalf("preview %q", prev)
	}
	if dig == "" || prev == "SECRET-BODY-SHOULD-NOT-PREVIEW" {
		t.Fatal("digest/preview leaked body")
	}
}
