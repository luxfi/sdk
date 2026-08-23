// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// docgen renders the Go SDK reference from source comments (godoc), one MDX
// page per package, for docs.lux.network. It PARSES the packages (go/parser +
// go/doc) rather than building them, so it is immune to build-graph skew.
//
//	go run ./tools/docgen <out-dir> [sdk-root]   # default root: .
package main

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const mod = "github.com/luxfi/sdk"

func main() {
	out, root := "sdk-go-docs", "."
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if len(os.Args) > 2 {
		root = os.Args[2]
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	entries, _ := os.ReadDir(root)
	var pages []string
	for _, e := range entries {
		if !e.IsDir() || skip(e.Name()) {
			continue
		}
		if md, ok := genPkg(filepath.Join(root, e.Name()), e.Name()); ok {
			os.WriteFile(filepath.Join(out, e.Name()+".mdx"), []byte(md), 0o644)
			pages = append(pages, e.Name())
		}
	}
	os.WriteFile(filepath.Join(out, "meta.json"),
		[]byte(`{"pages":["`+strings.Join(pages, `","`)+`"]}`+"\n"), 0o644)
	fmt.Printf("sdk docgen: %d packages → %s\n", len(pages), out)
}

func skip(n string) bool {
	switch n {
	case "internal", "examples", "cmd", "vendor", "docs", "bin", "testdata", "scripts", "tools", "mocks":
		return true
	}
	return strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") ||
		strings.HasSuffix(n, "test") || strings.Contains(n, "mock")
}

func genPkg(dir, name string) (string, bool) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil || len(pkgs) == 0 {
		return "", false
	}
	var p *ast.Package
	var pname string
	for n, pk := range pkgs {
		if strings.HasSuffix(n, "_test") || n == "main" {
			continue
		}
		p, pname = pk, n
		break
	}
	if p == nil {
		return "", false
	}
	d := doc.New(p, mod+"/"+name, 0)
	if d.Doc == "" && len(d.Funcs) == 0 && len(d.Types) == 0 {
		return "", false // nothing exported worth a page
	}
	syn := doc.Synopsis(d.Doc)
	if syn == "" {
		syn = "Package " + pname
	}
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: %s\ndescription: %s\n---\n\n", pname, inline(syn))
	b.WriteString("{/* Generated from godoc by tools/docgen — edit the Go source, not this file. */}\n\n")
	fmt.Fprintf(&b, "```go\nimport \"%s/%s\"\n```\n\n", mod, name)
	if d.Doc != "" {
		b.WriteString(prose(d.Doc) + "\n\n")
	}
	if len(d.Funcs) > 0 {
		b.WriteString("## Functions\n\n")
		for _, f := range d.Funcs {
			fmt.Fprintf(&b, "### %s\n\n```go\n%s\n```\n\n", f.Name, sig(fset, f.Decl))
			if f.Doc != "" {
				b.WriteString(prose(f.Doc) + "\n\n")
			}
		}
	}
	if len(d.Types) > 0 {
		b.WriteString("## Types\n\n")
		for _, t := range d.Types {
			fmt.Fprintf(&b, "### %s\n\n", t.Name)
			if t.Doc != "" {
				b.WriteString(prose(t.Doc) + "\n\n")
			}
			for _, f := range t.Funcs {
				fmt.Fprintf(&b, "```go\n%s\n```\n\n", sig(fset, f.Decl))
			}
			for _, m := range t.Methods {
				fmt.Fprintf(&b, "```go\n%s\n```\n\n", sig(fset, m.Decl))
			}
		}
	}
	return b.String(), true
}

// sig prints a func signature (no body) from its declaration.
func sig(fset *token.FileSet, d *ast.FuncDecl) string {
	body := d.Body
	d.Body = nil
	var buf strings.Builder
	printer.Fprint(&buf, fset, d)
	d.Body = body
	return strings.TrimSpace(buf.String())
}

// prose makes doc comments MDX-safe outside code fences.
func prose(s string) string {
	lines := strings.Split(s, "\n")
	fenced := false
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		ln = strings.ReplaceAll(ln, "<", "&lt;")
		lines[i] = strings.ReplaceAll(ln, "{", "&#123;")
	}
	return strings.Join(lines, "\n")
}

func inline(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), `"`, `'`)
	return strings.ReplaceAll(s, "<", "&lt;")
}
