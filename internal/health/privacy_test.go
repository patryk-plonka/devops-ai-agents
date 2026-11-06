package health

import "testing"

func TestSafeText(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		count       int
	}{
		{"Client [east]", "Client [east]", 0},
		{"\x1b[31mclient\x1b[0m\n\x00", "client", 0},
		{"\x1b]0;hostile title\x07client", "client", 0},
		{"Bearer canary", "[REDACTED]", 1},
		{"Bearer\ncanary", "[REDACTED]", 1},
		{"https://canary@example.com", "[REDACTED]", 1},
		{"eyJhbGciOiJub25lIn0.eyJzdWIiOiJ0ZXN0In0.signature", "[REDACTED]", 1},
		{"https://user:canary@example.com", "[REDACTED]", 1},
		{"https://storage.example/?sv=1&sig=canary", "[REDACTED]", 1},
		{"AccountKey=canary", "[REDACTED]", 1},
		{"-----BEGIN RSA PRIVATE KEY-----", "[REDACTED]", 1},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, n := SafeText(tc.input)
			if got != tc.want || n != tc.count {
				t.Fatalf("got %q (%d), want %q (%d)", got, n, tc.want, tc.count)
			}
		})
	}
}

func TestSafeTagsCredentialKeysAndCollisions(t *testing.T) {
	for _, key := range []string{"password", "db.password", "CLIENT-SECRET", "access_token", "refresh token", "Authorization", "API key", "account_key", "connection-string", "private key", "SAS", "storageSasToken", "SharedAccessSignature", "SAS-signature", "azureSas"} {
		tags, n := SafeTags(map[string]*string{key: ptr("canary")})
		if tags[key] != "[REDACTED]" || n != 1 {
			t.Errorf("key %s retained credential", key)
		}
	}
	tags, n := SafeTags(map[string]*string{"env": ptr("dev"), "e\x00nv": ptr("prod")})
	if tags["env"] != "[REDACTED]" || n != 1 {
		t.Fatal("normalization collision overwrote tag")
	}
}

func TestCredentialURLsAcrossSchemes(t *testing.T) {
	for _, input := range []string{
		"rediss://app:credential-canary@cache.example:6380",
		"redis://:credential-canary@cache.example",
		"postgresql://app:credential-canary@db.example/database",
		"postgresql+driver://app:credential-canary@db.example/database",
		"custom.v1-protocol://app:credential-canary@example.com",
		"HTTPS://app:credential-canary@example.com",
		"endpoint=postgresql://app:credential-canary@db.example/database",
		"re\u200bdiss://app:credential-canary@cache.example",
	} {
		got, n := SafeText(input)
		if got != "[REDACTED]" || n != 1 {
			t.Errorf("credential URL was retained for %q", input)
		}
		tags, count := SafeTags(map[string]*string{"endpoint": ptr(input)})
		if tags["endpoint"] != "[REDACTED]" || count != 1 {
			t.Error("credential tag URL was retained")
		}
	}
	for _, input := range []string{"redis://cache.example:6380", "postgresql://db.example/database", "https://example.com/?contact=user@example.com", "team@example.com"} {
		if got, n := SafeText(input); got != input || n != 0 {
			t.Errorf("ordinary text changed: %q", input)
		}
	}
}
