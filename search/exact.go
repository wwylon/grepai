package search

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ExactHit is one on-disk match. Results reflect the working tree, not the embed index.
type ExactHit struct {
	Project string `json:"project"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Text    string `json:"text"`
}

// ExactRoot is a workspace member to search on disk.
type ExactRoot struct {
	Name string
	Path string
}

func GrepRoots(pattern string, roots []ExactRoot, regex bool) ([]ExactHit, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	if _, err := exec.LookPath("rg"); err == nil {
		return grepRipgrep(pattern, roots, regex)
	}
	return grepWalk(pattern, roots, regex)
}

func grepRipgrep(pattern string, roots []ExactRoot, regex bool) ([]ExactHit, error) {
	var hits []ExactHit
	for _, root := range roots {
		args := []string{
			"--line-number", "--no-heading", "--color", "never",
			"--path-separator", "/",
			"-g", "!.git/", "-g", "!node_modules/",
		}
		if !regex {
			args = append(args, "-F")
		}
		args = append(args, "--", pattern, root.Path)
		cmd := exec.Command("rg", args...)
		out, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
				continue
			}
			return hits, fmt.Errorf("rg in %s: %w", root.Name, err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimRight(line, "\r")
			if line == "" {
				continue
			}
			hit, ok := parseRgLine(root, line)
			if ok {
				hits = append(hits, hit)
			}
		}
	}
	return hits, nil
}

func parseRgLine(root ExactRoot, line string) (ExactHit, bool) {
	// First ":digits:" after the path. With --path-separator /, Windows
	// drive letters (C:/...) do not steal the line-number colon.
	abs, n, text, ok := splitRgPathLine(line)
	if !ok {
		return ExactHit{}, false
	}
	rel, err := filepath.Rel(root.Path, filepath.FromSlash(abs))
	if err != nil {
		rel = abs
	}
	return ExactHit{
		Project: root.Name,
		Path:    filepath.ToSlash(rel),
		Line:    n,
		Text:    text,
	}, true
}

func splitRgPathLine(line string) (path string, n int, text string, ok bool) {
	for i := 0; i < len(line); i++ {
		if line[i] != ':' {
			continue
		}
		j := i + 1
		if j >= len(line) || line[j] < '0' || line[j] > '9' {
			continue
		}
		for j < len(line) && line[j] >= '0' && line[j] <= '9' {
			j++
		}
		if j >= len(line) || line[j] != ':' {
			continue
		}
		if _, err := fmt.Sscanf(line[i+1:j], "%d", &n); err != nil {
			return "", 0, "", false
		}
		return line[:i], n, line[j+1:], true
	}
	return "", 0, "", false
}

func grepWalk(pattern string, roots []ExactRoot, regex bool) ([]ExactHit, error) {
	var re *regexp.Regexp
	var err error
	if regex {
		re, err = regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
	}
	var hits []ExactHit
	for _, root := range roots {
		err := filepath.WalkDir(root.Path, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			name := d.Name()
			if d.IsDir() {
				if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			info, err := d.Info()
			if err != nil || info.Size() > 1*1024*1024 {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 64*1024), 1024*1024)
			lineNo := 0
			for sc.Scan() {
				lineNo++
				text := sc.Text()
				ok := false
				if regex {
					ok = re.MatchString(text)
				} else {
					ok = strings.Contains(text, pattern)
				}
				if !ok {
					continue
				}
				rel, relErr := filepath.Rel(root.Path, path)
				if relErr != nil {
					rel = path
				}
				hits = append(hits, ExactHit{
					Project: root.Name,
					Path:    filepath.ToSlash(rel),
					Line:    lineNo,
					Text:    text,
				})
			}
			return nil
		})
		if err != nil {
			return hits, err
		}
	}
	return hits, nil
}
