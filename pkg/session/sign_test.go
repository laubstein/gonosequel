package session

import "testing"

func TestSignIDVerifyRoundTrip(t *testing.T) {
	token := SignID("s3cr3t", "abc-123")

	id, ok := VerifyID("s3cr3t", token)
	if !ok {
		t.Fatalf("VerifyID(%q) = false, want true", token)
	}
	if id != "abc-123" {
		t.Errorf("VerifyID id = %q, want abc-123", id)
	}
}

func TestVerifyIDRejectsWrongSecret(t *testing.T) {
	token := SignID("s3cr3t", "abc-123")

	if _, ok := VerifyID("other-secret", token); ok {
		t.Errorf("VerifyID with wrong secret = true, want false")
	}
}

func TestVerifyIDRejectsTamperedID(t *testing.T) {
	token := SignID("s3cr3t", "abc-123")
	tampered := "xyz-999" + token[len("abc-123"):]

	if _, ok := VerifyID("s3cr3t", tampered); ok {
		t.Errorf("VerifyID with tampered id = true, want false")
	}
}

func TestVerifyIDRejectsMalformedToken(t *testing.T) {
	cases := []string{"", "no-dot-here", "abc-123.not-hex-zz"}
	for _, tc := range cases {
		if _, ok := VerifyID("s3cr3t", tc); ok {
			t.Errorf("VerifyID(%q) = true, want false", tc)
		}
	}
}
