// Command package builds release archives for every platform from any operating system:
//
//	go run ./scripts/package [-out dist] [-version 0.1.0]
//
// It exists because archives made with the tar of Git for Windows lose the executable bit, and a Linux
// or macOS user would then get "Permission denied". Here the mode of every file is set explicitly
// (0755 for the program), timestamps are fixed to the commit time so builds are reproducible, and a
// checksums.txt (sha256) is written next to the archives.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type target struct {
	goos, goarch string
	experimental string // "" when the platform is the main one
}

var targets = []target{
	{"windows", "amd64", ""},
	{"windows", "arm64", ""},
	{"linux", "amd64", "Linux build: experimental (see README-linux.md)"},
	{"linux", "arm64", "Linux build: experimental (see README-linux.md)"},
	{"darwin", "amd64", "macOS build: experimental, not yet run on a real Mac"},
	{"darwin", "arm64", "macOS build: experimental, not yet run on a real Mac"},
}

func main() {
	out := flag.String("out", "dist", "output folder")
	version := flag.String("version", "", "version to embed (default: 0.1.0-dev+<commit>)")
	only := flag.String("only", "", "comma-separated goos list (e.g. linux)")
	flag.Parse()
	if *version == "" {
		rev, _ := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
		*version = "0.1.0-dev+" + strings.TrimSpace(string(rev))
	}
	stamp := time.Now().UTC()
	if t, err := exec.Command("git", "log", "-1", "--format=%cI").Output(); err == nil {
		if p, err := time.Parse(time.RFC3339, strings.TrimSpace(string(t))); err == nil {
			stamp = p.UTC()
		}
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}
	var sums []string
	for _, t := range targets {
		if *only != "" && !strings.Contains(","+*only+",", ","+t.goos+",") {
			continue
		}
		name := fmt.Sprintf("catude_%s_%s", t.goos, t.goarch)
		exe := "catude"
		if t.goos == "windows" {
			exe += ".exe"
		}
		bin := filepath.Join(*out, name+"-"+exe)
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w -X main.version="+*version, "-o", bin, "./cmd/catude")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+t.goos, "GOARCH="+t.goarch)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fail(fmt.Errorf("build %s: %w", name, err))
		}
		files := []entry{{exe, bin, 0o755}, {"LICENSE", "LICENSE", 0o644}, {"README.md", "README.md", 0o644}}
		if t.goos == "linux" {
			files = append(files, entry{"README-linux.md", "docs/linux.md", 0o644})
		}
		if t.experimental != "" {
			note := filepath.Join(*out, name+"-EXPERIMENTAL.txt")
			os.WriteFile(note, []byte("catude "+*version+" for "+t.goos+"/"+t.goarch+"\n"+t.experimental+"\nRun ./catude install to put catude, ct and ctlite on your PATH.\n"), 0o644)
			files = append(files, entry{"EXPERIMENTAL.txt", note, 0o644})
			defer os.Remove(note)
		}
		archive := filepath.Join(*out, name+".tar.gz")
		if t.goos == "windows" {
			archive = filepath.Join(*out, name+".zip")
			err := writeZip(archive, name, files, stamp)
			check(err)
		} else {
			check(writeTarGz(archive, name, files, stamp))
		}
		os.Remove(bin)
		sum, err := fileSHA(archive)
		check(err)
		sums = append(sums, sum+"  "+filepath.Base(archive))
		fmt.Println("wrote", archive)
	}
	sort.Strings(sums)
	check(os.WriteFile(filepath.Join(*out, "checksums.txt"), []byte(strings.Join(sums, "\n")+"\n"), 0o644))
	fmt.Println("wrote", filepath.Join(*out, "checksums.txt"))
}

type entry struct {
	name, src string
	mode      int64
}

func writeTarGz(path, dir string, files []entry, stamp time.Time) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	gz.ModTime = stamp
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: dir + "/", Mode: 0o755, Typeflag: tar.TypeDir, ModTime: stamp, Format: tar.FormatPAX}); err != nil {
		return err
	}
	for _, e := range files {
		b, err := os.ReadFile(e.src)
		if err != nil {
			return err
		}
		h := &tar.Header{Name: dir + "/" + e.name, Mode: e.mode, Size: int64(len(b)), Typeflag: tar.TypeReg, ModTime: stamp, Format: tar.FormatPAX}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if _, err := tw.Write(b); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func writeZip(path, dir string, files []entry, stamp time.Time) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for _, e := range files {
		b, err := os.ReadFile(e.src)
		if err != nil {
			return err
		}
		h := &zip.FileHeader{Name: dir + "/" + e.name, Method: zip.Deflate, Modified: stamp}
		h.SetMode(os.FileMode(e.mode))
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	return zw.Close()
}

func fileSHA(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func check(err error) {
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "package:", err)
	os.Exit(1)
}
