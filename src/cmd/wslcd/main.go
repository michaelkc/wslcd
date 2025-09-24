package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
		return
	}

	arg := os.Args[1]
	cwd, err := os.Getwd()
	if err != nil {
		failf("error: unable to get current working directory: %v", err)
	}

	home := os.Getenv("HOME")

	target, err := ResolveTarget(arg, cwd, home)
	if err != nil {
		failf("%v", err)
	}

	// Print the resolved path for the shell wrapper to cd into.
	fmt.Println(target)
}

func usage() {
	fmt.Fprintf(os.Stderr, `wslcd - resolve Linux or Windows-style paths for cd

Usage:
  wslcd <path>

Examples:
  wslcd /var/log
  wslcd ../src
  wslcd ~/projects
  wslcd "C:\\Users\\me\\Documents"
  wslcd "D:/Work/Repo"
  wslcd c:JunkProjectsMyRepo   # collapsed Windows path without separators

This program prints the resolved target directory. Use a shell wrapper to actually cd:
  wslcd() { local t; t="$(command wslcd "$@")" || return; [ -z "$t" ] && return; cd -- "$t"; }
`)
}

func failf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

// ResolveTarget resolves arg either as a Linux path or a Windows path mapped under /mnt/<drive>.
// Returns an absolute path to an existing directory.
func ResolveTarget(arg, cwd, home string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", errors.New("error: missing target directory")
	}

	// Standard Windows path (e.g., C:\\ or C:/)
	if isWindowsPath(arg) {
		return resolveWindowsPath(arg)
	}
	// Collapsed Windows path like "C:FooBarBaz" (shell ate backslashes)
	if looksLikeWindowsDriveNoSlash(arg) {
		return resolveWindowsPathCollapsed(arg)
	}

	// Linux path semantics
	p, err := resolveLinuxLike(arg, cwd, home)
	if err != nil {
		return "", err
	}
	// verify dir
	info, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("error: %s", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("error: not a directory: %s", p)
	}
	return p, nil
}

// resolveLinuxLike resolves ~, relative, and cleans the path.
func resolveLinuxLike(arg, cwd, home string) (string, error) {
	p := arg
	// ~ or ~/...
	if p == "~" {
		if home == "" {
			return "", errors.New("error: HOME is not set")
		}
		p = home
	} else if strings.HasPrefix(p, "~/") {
		if home == "" {
			return "", errors.New("error: HOME is not set")
		}
		p = filepath.Join(home, p[2:])
	} else if !strings.HasPrefix(p, "/") {
		// relative
		p = filepath.Join(cwd, p)
	}
	return filepath.Clean(p), nil
}

// isWindowsPath detects drive-letter rooted paths like "C:\\..." or "d:/...".
func isWindowsPath(p string) bool {
	if len(p) < 3 {
		return false
	}
	// [A-Za-z]:[/\]
	r0 := rune(p[0])
	if !unicode.IsLetter(r0) {
		return false
	}
	if p[1] != ':' {
		return false
	}
	sep := p[2]
	return sep == '\\' || sep == '/'
}

// looksLikeWindowsDriveNoSlash detects inputs like "C:Something" where the path separators were lost.
func looksLikeWindowsDriveNoSlash(p string) bool {
	if len(p) < 3 {
		return false
	}
	if !unicode.IsLetter(rune(p[0])) || p[1] != ':' {
		return false
	}
	return p[2] != '\\' && p[2] != '/'
}

// resolveWindowsPath maps e.g. "C:\\Foo\\Bar" -> best matching "/mnt/c/Foo/Bar" using case-insensitive segment matching.
func resolveWindowsPath(win string) (string, error) {
	drive := unicode.ToLower(rune(win[0]))
	rest := win[2:] // starts with '\\' or '/'
	rest = strings.ReplaceAll(rest, "\\", "/")

	// Normalize segments and handle . and ..
	var segs []string
	for _, s := range strings.Split(rest, "/") {
		if s == "" { continue }
		if s == "." { continue }
		if s == ".." { if len(segs) > 0 { segs = segs[:len(segs)-1] }; continue }
		segs = append(segs, s)
	}

	mntRoot, err := pickCaseInsensitiveEntry("/mnt", string(drive))
	if err != nil {
		return "", fmt.Errorf("error: cannot locate /mnt/%c (drive mapping): %v", drive, err)
	}
	root := filepath.Join("/mnt", mntRoot)

	cands, err := exploreCandidates(root, segs)
	if err != nil { return "", err }
	if len(cands) == 0 {
		if len(segs) == 0 {
			info, err := os.Stat(root)
			if err != nil { return "", fmt.Errorf("error: %v", err) }
			if !info.IsDir() { return "", fmt.Errorf("error: not a directory: %s", root) }
			return root, nil
		}
		return "", fmt.Errorf("error: path does not exist (no case-insensitive match): %s", win)
	}

	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score { return cands[i].score > cands[j].score }
		return cands[i].fullPath < cands[j].fullPath
	})
	return cands[0].fullPath, nil
}

// resolveWindowsPathCollapsed greedily matches directory names as case-insensitive prefixes of the tail.
func resolveWindowsPathCollapsed(win string) (string, error) {
	drive := unicode.ToLower(rune(win[0]))
	tail := win[2:]

	mntRoot, err := pickCaseInsensitiveEntry("/mnt", string(drive))
	if err != nil {
		return "", fmt.Errorf("error: cannot locate /mnt/%c (drive mapping): %v", drive, err)
	}
	curr := filepath.Join("/mnt", mntRoot)

	tail = strings.TrimLeft(tail, "\\/")
	for {
		if len(tail) == 0 {
			info, err := os.Stat(curr)
			if err != nil { return "", fmt.Errorf("error: %v", err) }
			if !info.IsDir() { return "", fmt.Errorf("error: not a directory: %s", curr) }
			return curr, nil
		}

		if tail[0] == '/' || tail[0] == '\\' { tail = strings.TrimLeft(tail, "\\/"); continue }

		ents, err := os.ReadDir(curr)
		if err != nil { return "", fmt.Errorf("error: cannot read directory %s: %v", curr, err) }

		type cand struct { name string; plen int; score int }
		var ms []cand
		for _, e := range ents {
			n := e.Name()
			ln := len(n)
			if ln > len(tail) { continue }
			if !strings.EqualFold(tail[:ln], n) { continue }
			full := filepath.Join(curr, n)
			isDir, err := isDirFollowSymlink(full, e)
			if err != nil || !isDir { continue }
			ms = append(ms, cand{name: n, plen: ln, score: caseScore(tail[:ln], n)})
		}

		if len(ms) == 0 {
			return "", fmt.Errorf("error: cannot segment '%s' at '%s' under %s\nHint: quote the Windows path or use forward slashes (e.g., C:/...)", tail, argHead(tail), curr)
		}

		sort.SliceStable(ms, func(i, j int) bool {
			if ms[i].plen != ms[j].plen { return ms[i].plen > ms[j].plen }
			if ms[i].score != ms[j].score { return ms[i].score > ms[j].score }
			return ms[i].name < ms[j].name
		})

		chosen := ms[0]
		curr = filepath.Join(curr, chosen.name)
		tail = tail[chosen.plen:]
	}
}

func argHead(s string) string {
	if len(s) == 0 { return "" }
	if len(s) > 16 { return s[:16] + "..." }
	return s
}

func pickCaseInsensitiveEntry(dir, want string) (string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil { return "", err }
	wantLower := strings.ToLower(want)
	type pair struct { name string; score int }
	var matches []pair
	for _, e := range ents {
		n := e.Name()
		if strings.EqualFold(n, want) {
			matches = append(matches, pair{name: n, score: caseScore(want, n)})
		}
	}
	if len(matches) == 0 {
		candidate := filepath.Join(dir, wantLower)
		if st, err := os.Stat(candidate); err == nil && st.IsDir() { return wantLower, nil }
		return "", fmt.Errorf("no match for %s in %s", want, dir)
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score { return matches[i].score > matches[j].score }
		return matches[i].name < matches[j].name
	})
	return matches[0].name, nil
}

type candidate struct { fullPath string; score int }

func exploreCandidates(root string, segs []string) ([]candidate, error) {
	type state struct { dir string; idx int; score int }
	var results []candidate
	var dfs func(st state) error
	dfs = func(st state) error {
		if st.idx >= len(segs) {
			info, err := os.Stat(st.dir)
			if err != nil { return nil }
			if info.IsDir() { results = append(results, candidate{fullPath: st.dir, score: st.score}) }
			return nil
		}
		seg := segs[st.idx]
		ents, err := os.ReadDir(st.dir)
		if err != nil { return nil }
		type match struct { name string; score int; path string }
		var ms []match
		for _, e := range ents {
			n := e.Name()
			if !strings.EqualFold(n, seg) { continue }
			full := filepath.Join(st.dir, n)
			isDir, err := isDirFollowSymlink(full, e)
			if err != nil || !isDir { if st.idx == len(segs)-1 { continue }; continue }
			ms = append(ms, match{name: n, score: caseScore(seg, n), path: full})
		}
		if len(ms) == 0 { return nil }
		for _, m := range ms {
			if err := dfs(state{dir: m.path, idx: st.idx + 1, score: st.score + m.score}); err != nil { return err }
		}
		return nil
	}
	if len(segs) == 0 {
		if info, err := os.Stat(root); err == nil && info.IsDir() { results = append(results, candidate{fullPath: root, score: 0}) }
		return results, nil
	}
	if err := dfs(state{dir: root, idx: 0, score: 0}); err != nil { return nil, err }
	return results, nil
}

func isDirFollowSymlink(full string, de fs.DirEntry) (bool, error) {
	if de.IsDir() { return true, nil }
	info, err := os.Stat(full)
	if err != nil { return false, err }
	return info.IsDir(), nil
}

func caseScore(input, candidate string) int {
	inRunes := []rune(input)
	cRunes := []rune(candidate)
	n := len(inRunes)
	if len(cRunes) < n { n = len(cRunes) }
	score := 0
	for i := 0; i < n; i++ { if inRunes[i] == cRunes[i] { score++ } }
	return score
}
