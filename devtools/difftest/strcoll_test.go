package difftest

import (
	"context"
	"fmt"
	"testing"

	"github.com/k1LoW/glibctext/collate"
	"github.com/k1LoW/glibctext/collate/glibc2_41/en_us_utf8"
)

// targets lists each collation with the image its tables were generated from.
var targets = []struct {
	locale    string
	version   string
	image     string
	collation collate.Collation
}{
	{
		locale:    "en_US.UTF-8",
		version:   "2.41",
		image:     "postgres:18@sha256:74935e72241653ca55e0414067e6d8763aceb8a810eb51b452253ec3dcfc4336",
		collation: en_us_utf8.Collation,
	},
}

const seed = 20261008

func TestStrcoll(t *testing.T) {
	for _, tt := range targets {
		t.Run(fmt.Sprintf("%s/glibc%s", tt.locale, tt.version), func(t *testing.T) {
			ctx := context.Background()
			h := StartHarness(ctx, t, tt.image)
			ps := Corpus(seed)
			want, version, err := h.Strcoll(ctx, tt.locale, ps)
			if err != nil {
				t.Fatal(err)
			}
			if version != tt.version {
				t.Fatalf("harness loaded glibc %s, want %s", version, tt.version)
			}
			t.Logf("%d pairs compared against glibc %s", len(ps), version)
			failed := 0
			for i, p := range ps {
				got := tt.collation.Compare(p.A, p.B)
				// The exact value is checked, not only the sign, since the
				// port is meant to follow strcoll_l step by step.
				if got == int(want[i]) {
					continue
				}
				failed++
				if failed <= 50 {
					t.Errorf("Compare(%s, %s) = %d, strcoll_l = %d", quote(p.A), quote(p.B), got, want[i])
				}
			}
			if failed > 0 {
				t.Errorf("%d of %d pairs differ (seed %d)", failed, len(ps), seed)
			}
		})
	}
}

func quote(s string) string {
	return fmt.Sprintf("%+q", s)
}
