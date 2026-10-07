// Command gen extracts glibc's compiled LC_COLLATE data for a locale from a
// container image and writes the Go tables and the collation package.
//
//	go run ./gen -image debian:trixie@sha256:... -locale en_US.UTF-8
//
// The image must be Debian based (Debian or Ubuntu), since the glibc version is
// read from the dpkg status file. Nothing is run inside the image. The locale-archive and the dpkg status
// file are copied out of a created container, so the image can be of another
// architecture than the host.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/k1LoW/glibcollate/internal/localename"
	"github.com/k1LoW/glibcollate/internal/strcoll"
)

func main() {
	image := flag.String("image", "", "container image, pinned by digest")
	platform := flag.String("platform", "linux/amd64", "platform of the image")
	locale := flag.String("locale", "en_US.UTF-8", "locale name")
	out := flag.String("out", "..", "root directory of the glibcollate module")
	flag.Parse()
	if *image == "" {
		log.Fatal("-image is required")
	}
	if err := run(*image, *platform, *locale, *out); err != nil {
		log.Fatal(err)
	}
}

func run(image, platform, locale, out string) error {
	if !strings.Contains(image, "@sha256:") {
		return fmt.Errorf("image %s must be pinned by digest so that the output is reproducible", image)
	}
	tmp, err := os.MkdirTemp("", "glibcollate-gen")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if err := docker("pull", "--quiet", "--platform", platform, image); err != nil {
		return err
	}
	id, err := dockerOutput("create", "--platform", platform, image)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	defer docker("rm", id) //nolint:errcheck
	archivePath := filepath.Join(tmp, "locale-archive")
	statusPath := filepath.Join(tmp, "status")
	if err := docker("cp", id+":/usr/lib/locale/locale-archive", archivePath); err != nil {
		return err
	}
	if err := docker("cp", id+":/var/lib/dpkg/status", statusPath); err != nil {
		return err
	}

	archive, err := os.ReadFile(archivePath)
	if err != nil {
		return err
	}
	status, err := os.ReadFile(statusPath)
	if err != nil {
		return err
	}
	debVersion, err := dpkgVersion(status, "libc6")
	if err != nil {
		return err
	}
	glibcVersion, _, _ := strings.Cut(debVersion, "-")

	name := localename.Normalize(locale)
	lc, err := extractFromArchive(archive, name)
	if err != nil {
		return err
	}
	d, err := parseLCCollate(lc)
	if err != nil {
		return err
	}
	t := &strcoll.Table{NRules: d.NRules, Rulesets: string(d.Rulesets), TableMB: &d.TableMB, WeightMB: string(d.WeightMB), ExtraMB: string(d.ExtraMB), IndirectMB: d.IndirectMB}
	if err := strcoll.CheckBounded(t); err != nil {
		log.Printf("warning: glibc can read past the end of a string with this table, so some invalid input cannot be matched: %v", err)
	}
	sum := sha256.Sum256(lc)
	hash := hex.EncodeToString(sum[:])
	tablePkg := "lccollate_" + hash[:12]

	src, err := tableSource(tablePkg, hash, d)
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(out, "internal", "tables", tablePkg, "table.go"), src); err != nil {
		return err
	}

	pkg := localename.PackageName(locale)
	versionDir := "glibc" + strings.ReplaceAll(glibcVersion, ".", "_")
	src, err = collationSource(collationParams{
		Package:      pkg,
		Locale:       locale,
		ArchiveName:  name,
		GlibcVersion: glibcVersion,
		DebVersion:   debVersion,
		Image:        image,
		Platform:     platform,
		Hash:         hash,
		TablePkg:     tablePkg,
	})
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(out, versionDir, pkg, "collation.go"), src); err != nil {
		return err
	}
	log.Printf("%s glibc %s (Debian %s): LC_COLLATE sha256 %s", locale, glibcVersion, debVersion, hash)
	return nil
}

func dpkgVersion(status []byte, pkg string) (string, error) {
	for stanza := range strings.SplitSeq(string(status), "\n\n") {
		if !regexp.MustCompile(`(?m)^Package: ` + regexp.QuoteMeta(pkg) + `$`).MatchString(stanza) {
			continue
		}
		if m := regexp.MustCompile(`(?m)^Version: (\S+)$`).FindStringSubmatch(stanza); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("package %s not found in dpkg status", pkg)
}

func docker(args ...string) error {
	cmd := exec.Command("docker", args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func dockerOutput(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}
	return string(b), nil
}

func writeFile(path string, src []byte) error {
	// The paths are built from the -out flag and the locale name given by the
	// person running the generator, and the files are Go sources to commit.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec
		return err
	}
	return os.WriteFile(path, src, 0o644) //nolint:gosec
}

func gofmt(b *bytes.Buffer) ([]byte, error) {
	src, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated source: %w", err)
	}
	return src, nil
}
