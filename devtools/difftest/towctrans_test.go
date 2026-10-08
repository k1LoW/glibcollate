package difftest

import (
	"context"
	"fmt"
	"testing"

	"github.com/k1LoW/glibctext/ctype"
	"github.com/k1LoW/glibctext/ctype/glibc2_41/en_us_utf8"
)

// ctypeTargets lists each ctype with the image its tables were generated from.
var ctypeTargets = []struct {
	locale  string
	version string
	image   string
	ctype   ctype.Ctype
}{
	{
		locale:  "en_US.UTF-8",
		version: "2.41",
		image:   "postgres:18@sha256:74935e72241653ca55e0414067e6d8763aceb8a810eb51b452253ec3dcfc4336",
		ctype:   en_us_utf8.Ctype,
	},
}

// TestTowctrans compares every wide character of Unicode, not a sample.
func TestTowctrans(t *testing.T) {
	for _, tt := range ctypeTargets {
		t.Run(fmt.Sprintf("%s/glibc%s", tt.locale, tt.version), func(t *testing.T) {
			ctx := context.Background()
			h := StartHarness(ctx, t, tt.image)
			ms, version, err := h.Towctrans(ctx, tt.locale)
			if err != nil {
				t.Fatal(err)
			}
			if version != tt.version {
				t.Fatalf("harness loaded glibc %s, want %s", version, tt.version)
			}
			t.Logf("%d wide characters mapped by glibc %s", len(ms), version)
			failed := 0
			for _, m := range ms {
				r := rune(m.WC)                                                     //nolint:gosec // wint_t and rune share the bits
				up, low := uint32(tt.ctype.ToUpper(r)), uint32(tt.ctype.ToLower(r)) //nolint:gosec // as above
				if up == m.Upper && low == m.Lower {
					continue
				}
				failed++
				if failed <= 50 {
					t.Errorf("%#x: ToUpper = %#x, ToLower = %#x, towupper_l = %#x, towlower_l = %#x", m.WC, up, low, m.Upper, m.Lower)
				}
			}
			if failed > 0 {
				t.Errorf("%d of %d wide characters differ", failed, len(ms))
			}
		})
	}
}
