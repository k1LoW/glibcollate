package localename

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"en_US.UTF-8", "en_US.utf8"},
		{"en_US.utf8", "en_US.utf8"},
		{"en_US.Utf-8", "en_US.utf8"},
		{"de_DE.ISO-8859-1", "de_DE.iso88591"},
		{"de_DE.8859-1", "de_DE.iso88591"},
		{"de_DE.UTF-8@euro", "de_DE.utf8@euro"},
		{"ja_JP.EUC-JP", "ja_JP.eucjp"},
		{"en_US", "en_US"},
		{"C", "C"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPackageName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"en_US.UTF-8", "en_us_utf8"},
		{"ja_JP.EUC-JP", "ja_jp_eucjp"},
		{"de_DE.UTF-8@euro", "de_de_utf8_euro"},
	}
	for _, tt := range tests {
		if got := PackageName(tt.in); got != tt.want {
			t.Errorf("PackageName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
