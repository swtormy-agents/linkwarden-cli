package linkwarden

import "testing"

func TestValidateBaseURL(t *testing.T) {
	n, insecure, err := ValidateBaseURL("https://linkwarden.example/")
	if err != nil || insecure || n != "https://linkwarden.example" {
		t.Fatalf("got %q %v %v", n, insecure, err)
	}
	_, insecure, err = ValidateBaseURL("http://127.0.0.1:3000")
	if err != nil || !insecure {
		t.Fatalf("http: err=%v insecure=%v", err, insecure)
	}
	if _, _, err := ValidateBaseURL("ftp://x"); err == nil {
		t.Fatal("expected scheme error")
	}
	if _, _, err := ValidateBaseURL("https:///path"); err == nil {
		t.Fatal("expected host error")
	}
	if _, _, err := ValidateBaseURL("https://user:pass@host"); err == nil {
		t.Fatal("expected userinfo error")
	}
}
