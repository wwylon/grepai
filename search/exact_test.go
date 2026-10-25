package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGrepWalkFixedAndRegex(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	if err := os.WriteFile(filepath.Join(dirA, "a.go"), []byte("package frontend\nfunc Auth() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirB, "b.go"), []byte("package api\nfunc Health() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	roots := []ExactRoot{{Name: "frontend", Path: dirA}, {Name: "api", Path: dirB}}
	hits, err := grepWalk("Auth", roots, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Project != "frontend" {
		t.Fatalf("fixed hits = %+v", hits)
	}
	hits, err = grepWalk("func [AH]", roots, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("regex hits = %+v", hits)
	}
}

func TestSplitRgPathLineWindowsDrive(t *testing.T) {
	path, n, text, ok := splitRgPathLine(`D:/src/app.go:12:func Auth()`)
	if !ok || path != `D:/src/app.go` || n != 12 || text != "func Auth()" {
		t.Fatalf("got path=%q n=%d text=%q ok=%v", path, n, text, ok)
	}
	path, n, text, ok = splitRgPathLine(`/src/app.go:3:hello:world`)
	if !ok || path != `/src/app.go` || n != 3 || text != "hello:world" {
		t.Fatalf("posix got path=%q n=%d text=%q ok=%v", path, n, text, ok)
	}
}
