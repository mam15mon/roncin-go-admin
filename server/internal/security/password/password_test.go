package password

import "testing"

func TestHashAndVerify(t *testing.T) {
	encoded, err := Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	matched, err := Verify("correct horse battery staple", encoded)
	if err != nil || !matched {
		t.Fatalf("Verify() = %v, %v; want true, nil", matched, err)
	}
	matched, err = Verify("not-the-password", encoded)
	if err != nil || matched {
		t.Fatalf("Verify() wrong password = %v, %v; want false, nil", matched, err)
	}
}

func TestHashRejectsShortPassword(t *testing.T) {
	if _, err := Hash("too-short"); err == nil {
		t.Fatal("Hash() error = nil, want minimum-length error")
	}
}
