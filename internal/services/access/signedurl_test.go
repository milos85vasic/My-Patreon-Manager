package access

import (
	"testing"
	"time"
)

func TestSignedURLGenerator_GenerateAndVerify(t *testing.T) {
	g := NewSignedURLGenerator("test-secret", 1*time.Hour)

	url := g.GenerateSignedURL("content1", "sub1")
	if url == "" {
		t.Fatal("expected non-empty URL")
	}

	// Extract params from the URL
	token, sub, exp := ExtractTokenFromQuery(url[len("/download/content1?"):])

	subscriberID, expires, err := ParseSignedURLParams(token, sub, exp)
	if err != nil {
		t.Fatal(err)
	}
	if subscriberID != "sub1" {
		t.Errorf("expected sub1, got %s", subscriberID)
	}

	ok := g.VerifySignedURL(token, "content1", subscriberID, expires)
	if !ok {
		t.Error("expected valid signature")
	}
}

func TestSignedURLGenerator_VerifyExpired(t *testing.T) {
	g := NewSignedURLGenerator("test-secret", -1*time.Hour)

	url := g.GenerateSignedURL("content1", "sub1")
	token, sub, exp := ExtractTokenFromQuery(url[len("/download/content1?"):])

	subscriberID, expires, err := ParseSignedURLParams(token, sub, exp)
	if err != nil {
		t.Fatal(err)
	}

	ok := g.VerifySignedURL(token, "content1", subscriberID, expires)
	if ok {
		t.Error("expected expired URL to fail verification")
	}
}

func TestSignedURLGenerator_VerifyWrongToken(t *testing.T) {
	g := NewSignedURLGenerator("test-secret", 1*time.Hour)

	url := g.GenerateSignedURL("content1", "sub1")
	_, sub, exp := ExtractTokenFromQuery(url[len("/download/content1?"):])

	_, expires, err := ParseSignedURLParams("wrong-token", sub, exp)
	if err != nil {
		t.Fatal(err)
	}

	ok := g.VerifySignedURL("wrong-token", "content1", "sub1", expires)
	if ok {
		t.Error("expected wrong token to fail verification")
	}
}

func TestParseSignedURLParams_InvalidExpiry(t *testing.T) {
	_, _, err := ParseSignedURLParams("token", "sub", "not-a-number")
	if err == nil {
		t.Error("expected error for invalid expiry")
	}
}

func TestExtractTokenFromQuery(t *testing.T) {
	token, sub, exp := ExtractTokenFromQuery("token=abc&sub=sub1&exp=12345")
	if token != "abc" {
		t.Errorf("expected abc, got %s", token)
	}
	if sub != "sub1" {
		t.Errorf("expected sub1, got %s", sub)
	}
	if exp != "12345" {
		t.Errorf("expected 12345, got %s", exp)
	}

	// Empty query
	token, sub, exp = ExtractTokenFromQuery("")
	if token != "" || sub != "" || exp != "" {
		t.Error("expected empty results for empty query")
	}

	// Malformed parts (no =)
	token, sub, exp = ExtractTokenFromQuery("badpart&token=x")
	if token != "x" {
		t.Errorf("expected x, got %s", token)
	}
}
