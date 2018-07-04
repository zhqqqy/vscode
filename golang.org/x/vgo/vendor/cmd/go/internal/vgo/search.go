// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgo

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cmd/go/internal/base"
	"cmd/go/internal/cfg"
	"cmd/go/internal/imports"
	"cmd/go/internal/module"
	"cmd/go/internal/search"
)

func expandImportPaths(args []string) []string {
	var out []string
	for _, a := range args {
		if search.IsMetaPackage(a) {
			switch a {
			default:
				fmt.Fprintf(os.Stderr, "warning: %q matches no packages when using modules", a)
			case "all":
				out = append(out, AllPackages(a)...)
			}
			continue
		}
		if strings.Contains(a, "...") {
			if build.IsLocalImport(a) {
				out = append(out, search.AllPackagesInFS(a)...)
			} else {
				out = append(out, AllPackages(a)...)
			}
			continue
		}
		out = append(out, a)
	}
	return out
}

// AllPackages returns all the packages that can be found
// under the $GOPATH directories and $GOROOT matching pattern.
// The pattern is either "all" (all packages), "std" (standard packages),
// "cmd" (standard commands), or a path including "...".
func AllPackages(pattern string) []string {
	pkgs := MatchPackages(pattern)
	if len(pkgs) == 0 {
		fmt.Fprintf(os.Stderr, "warning: %q matched no packages\n", pattern)
	}
	return pkgs
}

// MatchPackages returns a list of package paths matching pattern
// (see go help packages for pattern syntax).
func MatchPackages(pattern string) []string {
	if pattern == "std" || pattern == "cmd" {
		return nil
	}
	if pattern == "all" {
		return MatchAll()
	}

	return matchPackages(pattern, buildList)
}

func matchPackages(pattern string, buildList []module.Version) []string {
	match := func(string) bool { return true }
	treeCanMatch := func(string) bool { return true }
	if !search.IsMetaPackage(pattern) {
		match = search.MatchPattern(pattern)
		treeCanMatch = search.TreeCanMatchPattern(pattern)
	}

	have := map[string]bool{
		"builtin": true, // ignore pseudo-package that exists only for documentation
	}
	if !cfg.BuildContext.CgoEnabled {
		have["runtime/cgo"] = true // ignore during walk
	}
	var pkgs []string

	for _, mod := range buildList {
		if !treeCanMatch(mod.Path) {
			continue
		}
		var root string
		if mod.Version == "" {
			root = ModRoot
		} else {
			var err error
			root, err = fetch(mod)
			if err != nil {
				base.Errorf("vgo: %v", err)
				continue
			}
		}
		root = filepath.Clean(root)

		filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			want := true
			// Avoid .foo, _foo, and testdata directory trees.
			_, elem := filepath.Split(path)
			if strings.HasPrefix(elem, ".") || strings.HasPrefix(elem, "_") || elem == "testdata" {
				want = false
			}

			name := mod.Path + filepath.ToSlash(path[len(root):])
			if !treeCanMatch(name) {
				want = false
			}

			if !fi.IsDir() {
				if fi.Mode()&os.ModeSymlink != 0 && want {
					if target, err := os.Stat(path); err == nil && target.IsDir() {
						fmt.Fprintf(os.Stderr, "warning: ignoring symlink %s\n", path)
					}
				}
				return nil
			}
			if !want {
				return filepath.SkipDir
			}

			if have[name] {
				return nil
			}
			have[name] = true
			if !match(name) {
				return nil
			}
			if _, _, err := imports.ScanDir(path, imports.Tags); err == imports.ErrNoGo {
				return nil
			}

			if elem == "vendor" {
				return filepath.SkipDir // ignore children
			}

			pkgs = append(pkgs, name)
			return nil
		})
	}
	return pkgs
}

// MatchAll returns a list of the packages matching the pattern "all".
// We redefine "all" to mean start with the packages in the current module
// and then follow imports into other modules to add packages imported
// (directly or indirectly) as part of builds in this module.
// It does not include packages in other modules that are not needed
// by builds of this module.
func MatchAll() []string {
	local := matchPackages("all", buildList[:1])
	ld := newLoader()
	ld.importList(local, levelTestRecursive)
	var all []string
	for _, pkg := range ld.importmap {
		if !search.IsStandardImportPath(pkg) {
			all = append(all, pkg)
		}
	}
	sort.Strings(all)
	return all
}
