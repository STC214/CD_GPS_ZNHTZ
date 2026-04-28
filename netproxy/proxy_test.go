package netproxy

import "testing"

func TestNormalizeServer(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "", want: ""},
		{raw: "127.0.0.1:7890", want: "http://127.0.0.1:7890"},
		{raw: "socks5://127.0.0.1:1080", want: "socks5://127.0.0.1:1080"},
		{raw: "socks4a://proxy.example:1080", want: "socks4a://proxy.example:1080"},
	}
	for _, tt := range tests {
		got, err := NormalizeServer(tt.raw)
		if err != nil {
			t.Fatalf("NormalizeServer(%q) error = %v", tt.raw, err)
		}
		if got != tt.want {
			t.Fatalf("NormalizeServer(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestNormalizeServerRejectsUnsupportedScheme(t *testing.T) {
	if _, err := NormalizeServer("ftp://127.0.0.1:21"); err == nil {
		t.Fatal("NormalizeServer() error = nil, want unsupported scheme error")
	}
}
