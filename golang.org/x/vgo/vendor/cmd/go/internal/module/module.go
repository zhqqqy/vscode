// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package module defines the module.Version type
// along with support code.
package module

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"cmd/go/internal/semver"
)

// A Version is defined by a module path and version pair.
type Version struct {
	Path    string
	Version string
}

// Check checks that a given module path, version pair is valid.
// In addition to the path being a valid module path
// and the version being a valid semantic version,
// the two must correspond.
// For example, the path "yaml/v2" only corresponds to
// semantic versions beginning with "v2.".
func Check(path, version string) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	if !semver.IsValid(version) {
		return fmt.Errorf("malformed semantic version %v", version)
	}
	_, pathVersion, _ := SplitPathVersion(path)
	pathVersion = strings.TrimPrefix(pathVersion, "/")
	vm := semver.Major(version)
	if vm == "v0" || vm == "v1" {
		vm = ""
	}
	if vm != pathVersion {
		if pathVersion == "" {
			pathVersion = "v0 or v1"
		}
		return fmt.Errorf("mismatched module path %v and version %v (want %v)", path, version, pathVersion)
	}
	return nil
}

// firstPathOK reports whether r can appear in the first element of a module path.
// The first element of the path must be an LDH domain name, at least for now.
func firstPathOK(r rune) bool {
	return r == '-' || r == '.' ||
		'0' <= r && r <= '9' ||
		'A' <= r && r <= 'Z' ||
		'a' <= r && r <= 'z'
}

// pathOK reports whether r can appear in a module path.
// Paths must avoid potentially problematic ASCII punctuation
// and control characters but otherwise can be any Unicode printable character,
// as defined by Go's IsPrint.
func pathOK(r rune) bool {
	if r < utf8.RuneSelf {
		return r == '+' || r == ',' || r == '-' || r == '.' || r == '/' || r == '_' || r == '~' ||
			'0' <= r && r <= '9' ||
			'A' <= r && r <= 'Z' ||
			'a' <= r && r <= 'z'
	}
	return unicode.IsPrint(r)
}

// CheckPath checks that a module path is valid.
func CheckPath(path string) error {
	if !utf8.ValidString(path) {
		return fmt.Errorf("malformed module path %q: invalid UTF-8", path)
	}
	if path == "" {
		return fmt.Errorf("malformed module path %q: empty string", path)
	}

	i := strings.Index(path, "/")
	if i < 0 {
		i = len(path)
	}
	if i == 0 {
		return fmt.Errorf("malformed module path %q: leading slash", path)
	}
	if !strings.Contains(path[:i], ".") {
		return fmt.Errorf("malformed module path %q: missing dot in first path element", path)
	}
	if path[i-1] == '.' {
		return fmt.Errorf("malformed module path %q: trailing dot in first path element", path)
	}
	if path[0] == '.' {
		return fmt.Errorf("malformed module path %q: leading dot in first path element", path)
	}
	if path[0] == '-' {
		return fmt.Errorf("malformed module path %q: leading dash in first path element", path)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("malformed module path %q: double dot", path)
	}
	if strings.Contains(path, "//") {
		return fmt.Errorf("malformed module path %q: double slash", path)
	}
	for _, r := range path[:i] {
		if !firstPathOK(r) {
			return fmt.Errorf("malformed module path %q: invalid char %q in first path element", path, r)
		}
	}
	if path[len(path)-1] == '/' {
		return fmt.Errorf("malformed module path %q: trailing slash", path)
	}
	for _, r := range path {
		if !pathOK(r) {
			return fmt.Errorf("malformed module path %q: invalid char %q", path, r)
		}
	}
	if _, _, ok := SplitPathVersion(path); !ok {
		return fmt.Errorf("malformed module path %q: invalid version %s", path, path[strings.LastIndex(path, "/")+1:])
	}
	return nil
}

// SplitPathVersion returns pathPrefix and version such that pathPrefix+pathMajor == path
// and version is either empty or "/vN" for N >= 2.
func SplitPathVersion(path string) (pathPrefix, pathMajor string, ok bool) {
	i := len(path)
	dot := false
	for i > 0 && ('0' <= path[i-1] && path[i-1] <= '9' || path[i-1] == '.') {
		if path[i-1] == '.' {
			dot = true
		}
		i--
	}
	if i <= 1 || path[i-1] != 'v' || path[i-2] != '/' {
		return path, "", true
	}
	pathPrefix, pathMajor = path[:i-2], path[i-2:]
	if dot || len(pathMajor) <= 2 || pathMajor[2] == '0' || pathMajor == "/v1" {
		return path, "", false
	}
	return pathPrefix, pathMajor, true
}

func MatchPathMajor(v, pathMajor string) bool {
	m := semver.Major(v)
	if pathMajor == "" {
		return m == "v0" || m == "v1"
	}
	return pathMajor[0] == '/' && m == pathMajor[1:]
}
