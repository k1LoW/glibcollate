package difftest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
)

// Harness runs the C programs in harness on an image.
type Harness struct {
	c testcontainers.Container
}

// StartHarness builds the harness on top of image and starts it.
// It skips the test when Docker is not available.
func StartHarness(ctx context.Context, t *testing.T, image string) *Harness {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)
	_, file, _, _ := runtime.Caller(0)
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:   filepath.Join(filepath.Dir(file), "harness"),
				BuildArgs: map[string]*string{"IMAGE": &image},
				KeepImage: true,
			},
		},
		Started: true,
	})
	testcontainers.CleanupContainer(t, c)
	if err != nil {
		t.Fatal(err)
	}
	return &Harness{c: c}
}

// Strcoll returns what strcoll_l returns for each pair under locale, and the
// glibc version the harness loaded.
func (h *Harness) Strcoll(ctx context.Context, locale string, ps []Pair) ([]int32, string, error) {
	var in bytes.Buffer
	for _, p := range ps {
		for _, s := range []string{p.A, p.B} {
			_ = binary.Write(&in, binary.LittleEndian, uint32(len(s))) //nolint:gosec // corpus strings are short
			in.WriteString(s)
		}
	}
	if err := h.c.CopyToContainer(ctx, in.Bytes(), "/tmp/in", 0o644); err != nil {
		return nil, "", err
	}
	code, r, err := h.c.Exec(ctx, []string{"sh", "-c", "strcoll '" + locale + "' < /tmp/in > /tmp/out"}, tcexec.Multiplexed())
	if err != nil {
		return nil, "", err
	}
	if code != 0 {
		msg, _ := io.ReadAll(r)
		return nil, "", fmt.Errorf("harness exited with %d: %s", code, msg)
	}
	rc, err := h.c.CopyFileFromContainer(ctx, "/tmp/out")
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()
	br := bufio.NewReader(rc)
	version, err := br.ReadString('\n')
	if err != nil {
		return nil, "", err
	}
	out := make([]int32, len(ps))
	if err := binary.Read(br, binary.LittleEndian, out); err != nil {
		return nil, "", fmt.Errorf("read results: %w", err)
	}
	return out, version[:len(version)-1], nil
}

// extraWC are the values past Unicode that harness/towctrans.c maps after
// 0 to 0x10ffff.
var extraWC = []uint32{0x110000, 0x7fffffff, 0x80000000, 0xfffffffe, 0xffffffff}

// Mapping is what towupper_l and towlower_l return for one wide character.
type Mapping struct {
	WC    uint32
	Upper uint32
	Lower uint32
}

// Towctrans returns what towupper_l and towlower_l return under locale for
// every wide character from 0 to 0x10ffff and for extraWC, and the glibc
// version the harness loaded.
func (h *Harness) Towctrans(ctx context.Context, locale string) ([]Mapping, string, error) {
	code, r, err := h.c.Exec(ctx, []string{"sh", "-c", "towctrans '" + locale + "' > /tmp/out"}, tcexec.Multiplexed())
	if err != nil {
		return nil, "", err
	}
	if code != 0 {
		msg, _ := io.ReadAll(r)
		return nil, "", fmt.Errorf("harness exited with %d: %s", code, msg)
	}
	rc, err := h.c.CopyFileFromContainer(ctx, "/tmp/out")
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()
	br := bufio.NewReader(rc)
	version, err := br.ReadString('\n')
	if err != nil {
		return nil, "", err
	}
	wcs := make([]uint32, 0, 0x110000+len(extraWC))
	for wc := range uint32(0x110000) {
		wcs = append(wcs, wc)
	}
	wcs = append(wcs, extraWC...)
	out := make([]uint32, 2*len(wcs))
	if err := binary.Read(br, binary.LittleEndian, out); err != nil {
		return nil, "", fmt.Errorf("read results: %w", err)
	}
	if _, err := br.ReadByte(); err != io.EOF {
		return nil, "", errors.New("harness wrote more results than expected")
	}
	ms := make([]Mapping, len(wcs))
	for i, wc := range wcs {
		ms[i] = Mapping{WC: wc, Upper: out[2*i], Lower: out[2*i+1]}
	}
	return ms, version[:len(version)-1], nil
}
