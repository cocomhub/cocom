// Copyright 2026 The Cocomhub Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ConfigEntry represents a single configuration key with its metadata.
type ConfigEntry struct {
	Key          string
	DefaultValue string
	Type         string
	Description  string
	SourceFile   string
	SourceLine   int
	HasSetDef    bool
	IsDeprecated bool
}

// GetCall records a viper.Get* call.
type GetCall struct {
	Key        string
	Type       string
	SourceFile string
	SourceLine int
}

var (
	projectDir string
	gitHash    string
	entries    = make(map[string]*ConfigEntry)
	getCalls   = make(map[string][]*GetCall)
	constMap   = make(map[string]string)
	prefixKeys = make(map[string]bool)
	allKeys    []string
)

func main() {
	output := flag.String("o", "docs/config.md", "Output file path")
	flag.Parse()

	execDir, _ := os.Getwd()
	projectDir = execDir
	// Walk up from cwd until we find go.mod to determine project root
	for dir := execDir; dir != "." && dir != "/" && dir != filepath.VolumeName(dir)+"\\"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			projectDir = dir
			break
		}
		// Stop at filesystem root
		if dir == filepath.Dir(dir) {
			break
		}
	}

	gitHash = getGitHash()
	scanDir(projectDir)
	generate(*output)
	fmt.Println("Config doc generated at", *output)
	reportWarnings()
}

func getGitHash() string {
	gitPath := filepath.Join(projectDir, ".git")
	fi, err := os.Stat(gitPath)
	if err != nil {
		return "unknown"
	}
	if fi.IsDir() {
		return readHeadFile(gitPath)
	}
	// Worktree: .git is a file with "gitdir: <path>"
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "unknown"
	}
	line := strings.TrimSpace(string(data))
	if after, ok := strings.CutPrefix(line, "gitdir: "); ok {
		mainGitDir := after
		// In worktrees, HEAD is relative to the worktree git dir, not the main repo
		return readHeadFile(mainGitDir)
	}
	return "unknown"
}

func readHeadFile(gitDir string) string {
	headContent, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "unknown"
	}
	ref := strings.TrimSpace(string(headContent))
	if !strings.HasPrefix(ref, "ref: ") {
		return ref
	}
	// Try to read the ref file
	refPath := filepath.Join(gitDir, strings.TrimPrefix(ref, "ref: "))
	if data, err := os.ReadFile(refPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	// For worktree-specific branches, the ref may be under worktrees/<name>/
	if filepath.Base(gitDir) == "worktrees" {
		return strings.TrimPrefix(ref, "ref: refs/heads/")
	}
	return strings.TrimPrefix(ref, "ref: refs/heads/")
}

func scanDir(dir string) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := info.Name()
			if base == ".claude" || base == ".trae" || base == ".cursor" || base == "vendor" || base == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		processFile(path)
		return nil
	})
}

func processFile(path string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return
	}

	relPath, _ := filepath.Rel(projectDir, path)
	relPath = filepath.ToSlash(relPath)

	collectConsts(f, fset)
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			processCallExpr(call, fset, relPath)
		}
		return true
	})
	collectConfigDocComments(f, fset, relPath)
}

func collectConsts(f *ast.File, fset *token.FileSet) {
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valSpec.Names {
				if i < len(valSpec.Values) {
					val := exprString(valSpec.Values[i])
					if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) {
						constMap[name.Name] = strings.Trim(val, `"`)
					}
				}
			}
		}
	}
}

func processCallExpr(call *ast.CallExpr, fset *token.FileSet, relPath string) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "viper" {
		return
	}
	method := sel.Sel.Name
	pos := fset.Position(call.Pos())

	if method == "SetDefault" && len(call.Args) >= 2 {
		key := extractKey(call.Args[0])
		if key == "" {
			return
		}
		defVal := exprString(call.Args[1])
		valType := inferType(call.Args[1])

		entry := getOrCreateEntry(key)
		entry.DefaultValue = defVal
		entry.Type = valType
		entry.SourceFile = relPath
		entry.SourceLine = pos.Line
		entry.HasSetDef = true
		return
	}

	if strings.HasPrefix(method, "Get") && len(call.Args) >= 1 {
		key := extractKey(call.Args[0])
		if key == "" {
			if binary, ok := call.Args[0].(*ast.BinaryExpr); ok && binary.Op == token.ADD {
				if ident, ok := binary.X.(*ast.Ident); ok {
					rhs := exprString(binary.Y)
					rhs = strings.Trim(rhs, `"`)
					if rhs != "" {
						prefixKeys[ident.Name] = true
					}
				}
			}
			return
		}

		getType := method[3:]
		gc := &GetCall{
			Key:        key,
			Type:       getType,
			SourceFile: relPath,
			SourceLine: pos.Line,
		}
		getCalls[key] = append(getCalls[key], gc)
		getOrCreateEntry(key)
	}
}

func getOrCreateEntry(key string) *ConfigEntry {
	if e, ok := entries[key]; ok {
		return e
	}
	e := &ConfigEntry{Key: key}
	entries[key] = e
	allKeys = append(allKeys, key)
	return e
}

func extractKey(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			return strings.Trim(v.Value, `"`)
		}
	case *ast.Ident:
		if val, ok := constMap[v.Name]; ok {
			return val
		}
	}
	return ""
}

func exprString(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		return v.Value
	case *ast.Ident:
		if val, ok := constMap[v.Name]; ok {
			return `"` + val + `"`
		}
		return v.Name
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		fun := exprString(v.Fun)
		args := make([]string, len(v.Args))
		for i, arg := range v.Args {
			args[i] = exprString(arg)
		}
		return fun + "(" + strings.Join(args, ", ") + ")"
	case *ast.CompositeLit:
		typ := exprString(v.Type)
		elts := make([]string, len(v.Elts))
		for i, elt := range v.Elts {
			elts[i] = exprString(elt)
		}
		if len(elts) == 0 {
			return typ + "{}"
		}
		return typ + "{" + strings.Join(elts, ", ") + "}"
	case *ast.BinaryExpr:
		return exprString(v.X) + " " + v.Op.String() + " " + exprString(v.Y)
	case *ast.ParenExpr:
		return "(" + exprString(v.X) + ")"
	case *ast.UnaryExpr:
		return v.Op.String() + exprString(v.X)
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	case *ast.KeyValueExpr:
		return exprString(v.Key) + ": " + exprString(v.Value)
	case *ast.IndexExpr:
		return exprString(v.X) + "[" + exprString(v.Index) + "]"
	case *ast.SliceExpr:
		low := ""
		if v.Low != nil {
			low = exprString(v.Low)
		}
		high := ""
		if v.High != nil {
			high = exprString(v.High)
		}
		return exprString(v.X) + "[" + low + ":" + high + "]"
	case *ast.MapType:
		return "map[" + exprString(v.Key) + "]" + exprString(v.Value)
	case *ast.ArrayType:
		if v.Len == nil {
			return "[]" + exprString(v.Elt)
		}
		return "[" + exprString(v.Len) + "]" + exprString(v.Elt)
	case *ast.InterfaceType:
		return "any"
	}
	return fmt.Sprintf("%T", expr)
}

func inferType(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		switch v.Kind {
		case token.STRING:
			return "string"
		case token.INT:
			return "int"
		case token.FLOAT:
			return "float64"
		case token.CHAR:
			return "string"
		}
	case *ast.Ident:
		if v.Name == "true" || v.Name == "false" {
			return "bool"
		}
		if val, ok := constMap[v.Name]; ok {
			if val == "true" || val == "false" {
				return "bool"
			}
			return "string"
		}
		return "string"
	case *ast.CompositeLit:
		switch t := v.Type.(type) {
		case *ast.ArrayType:
			return "[]" + exprString(t.Elt)
		case *ast.MapType:
			return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
		case *ast.Ident:
			return t.Name
		case *ast.SelectorExpr:
			return exprString(t)
		default:
			return exprString(v.Type)
		}
	case *ast.SelectorExpr:
		return exprString(v)
	case *ast.CallExpr:
		fun := exprString(v.Fun)
		if fun == "string" {
			return "string"
		}
		if strings.HasPrefix(fun, "time.") {
			return "duration"
		}
		return exprString(v)
	case *ast.BinaryExpr:
		return "duration"
	case *ast.ParenExpr:
		return inferType(v.X)
	case *ast.UnaryExpr:
		if v.Op == token.SUB {
			if lit, ok := v.X.(*ast.BasicLit); ok && lit.Kind == token.INT {
				return "int"
			}
		}
	}
	return "any"
}

func collectConfigDocComments(f *ast.File, fset *token.FileSet, relPath string) {
	configDocRe := regexp.MustCompile(`^//\s*config-doc:\s+(\S+)\s+(.*)$`)

	for _, cg := range f.Comments {
		for _, comment := range cg.List {
			matches := configDocRe.FindStringSubmatch(comment.Text)
			if matches == nil {
				continue
			}
			key := matches[1]
			desc := strings.TrimSpace(matches[2])

			entry := getOrCreateEntry(key)
			entry.Description = desc
			if strings.Contains(desc, "已废弃") || strings.Contains(desc, "deprecated") {
				entry.IsDeprecated = true
			}
		}
	}
}

func generate(output string) {
	var b strings.Builder

	now := time.Now().Format(time.RFC3339)

	b.WriteString("# Configuration Reference\n\n")
	b.WriteString("> Auto-generated by tools/config-doc-gen. DO NOT EDIT MANUALLY.\n")
	b.WriteString(fmt.Sprintf("> Generated at: %s\n", now))
	b.WriteString(fmt.Sprintf("> Git commit: %s\n\n", gitHash))

	b.WriteString("## Environment Variables\n\n")
	b.WriteString("All configuration keys can be set via environment variables with the `COCOM_` prefix.\n")
	b.WriteString("Nested keys use `_` as separator. Example:\n\n")
	b.WriteString("```bash\n")
	b.WriteString("export COCOM_MONGO_HOST=mongo.example.com:27017\n")
	b.WriteString("export COCOM_SERVER_PORT=8080\n")
	b.WriteString("export COCOM_LOG_ENABLE_CONSOLE=false\n")
	b.WriteString("```\n\n")
	b.WriteString("> NOTE: Existing `viper.AutomaticEnv()` (without prefix) still works for compatibility.\n")
	b.WriteString("> Use the `COCOM_` prefix for new deployments.\n\n")

	sort.Strings(allKeys)

	b.WriteString("## Configuration Keys\n\n")
	b.WriteString("| Key | Type | Default | Description | Source |\n")
	b.WriteString("|-----|------|---------|-------------|--------|\n")

	for _, key := range allKeys {
		entry := entries[key]
		if entry == nil || !entry.HasSetDef {
			continue
		}
		def := entry.DefaultValue
		if def == "" {
			def = "—"
		}
		desc := entry.Description
		if desc == "" {
			desc = "*No description*"
		}
		dep := ""
		if entry.IsDeprecated {
			dep = " (deprecated)"
		}
		src := fmt.Sprintf("`%s:%d`", entry.SourceFile, entry.SourceLine)

		b.WriteString(fmt.Sprintf("| `%s`%s | %s | `%s` | %s | %s |\n",
			entry.Key, dep, entry.Type, def, desc, src))
	}

	b.WriteString("\n## Keys with Default Values (grouped by prefix)\n\n")

	groups := groupByPrefix(allKeys)
	prefixes := make([]string, 0, len(groups))
	for p := range groups {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	for _, prefix := range prefixes {
		label := prefix
		if label == "" {
			label = "(root)"
		}
		b.WriteString(fmt.Sprintf("### %s\n\n", label))
		for _, key := range groups[prefix] {
			entry := entries[key]
			if entry == nil || !entry.HasSetDef {
				continue
			}
			dep := ""
			if entry.IsDeprecated {
				dep = " *(deprecated)*"
			}
			b.WriteString(fmt.Sprintf("- `%s` (%s): %s Default: `%s`. Defined in `%s:%d`.%s\n",
				entry.Key, entry.Type, entry.Description, entry.DefaultValue,
				entry.SourceFile, entry.SourceLine, dep))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Keys without Default Values\n\n")
	b.WriteString("Keys referenced in code but without viper.SetDefault — they use Go zero values if unset.\n")
	b.WriteString("Note: Keys migrated to config.Get().Field no longer appear here (they use Config struct defaults).\n\n")

	noDefaultKeys := make([]string, 0)
	for key := range getCalls {
		entry := entries[key]
		if entry != nil && entry.HasSetDef {
			continue
		}
		noDefaultKeys = append(noDefaultKeys, key)
	}
	sort.Strings(noDefaultKeys)

	if len(noDefaultKeys) == 0 {
		b.WriteString("*No keys in this category.*\n\n")
	} else {
		for _, key := range noDefaultKeys {
			calls := getCalls[key]
			sources := make([]string, len(calls))
			for i, c := range calls {
				sources[i] = fmt.Sprintf("`%s:%d`", c.SourceFile, c.SourceLine)
			}
			srcStr := strings.Join(sources, ", ")
			b.WriteString(fmt.Sprintf("- `%s`: Used in %s\n", key, srcStr))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Generation Report\n\n")
	b.WriteString(fmt.Sprintf("- Total configuration keys with SetDefault: %d\n", countWithSetDef()))
	b.WriteString(fmt.Sprintf("- Keys with config-doc description: %d\n", countWithDesc()))
	b.WriteString(fmt.Sprintf("- Keys without config-doc description: %d\n", countWithoutDesc()))
	b.WriteString(fmt.Sprintf("- Keys used via Get* without SetDefault: %d\n", len(noDefaultKeys)))

	_ = os.MkdirAll(filepath.Dir(output), 0o755)
	_ = os.WriteFile(output, []byte(b.String()), 0o644)
}

func groupByPrefix(keys []string) map[string][]string {
	groups := make(map[string][]string)
	for _, key := range keys {
		entry := entries[key]
		if entry == nil || !entry.HasSetDef {
			continue
		}
		prefix := extractPrefix(key)
		groups[prefix] = append(groups[prefix], key)
	}
	return groups
}

func extractPrefix(key string) string {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) >= 2 {
		return parts[0] + ".*"
	}
	if len(parts) == 1 {
		return parts[0] + ".*"
	}
	return ""
}

func countWithSetDef() int {
	n := 0
	for _, e := range entries {
		if e.HasSetDef {
			n++
		}
	}
	return n
}

func countWithDesc() int {
	n := 0
	for _, e := range entries {
		if e.Description != "" {
			n++
		}
	}
	return n
}

func countWithoutDesc() int {
	n := 0
	for _, e := range entries {
		if e.HasSetDef && e.Description == "" {
			n++
		}
	}
	return n
}

func reportWarnings() {
	fmt.Println("\n=== Warnings ===")
	fmt.Println("Keys with viper.SetDefault but NO config-doc comment:")
	for _, key := range allKeys {
		entry := entries[key]
		if entry != nil && entry.HasSetDef && entry.Description == "" {
			fmt.Printf("  - %s (in %s:%d)\n", entry.Key, entry.SourceFile, entry.SourceLine)
		}
	}

	fmt.Println("\nKeys used via viper.Get* but WITHOUT viper.SetDefault:")
	for _, key := range allKeys {
		entry := entries[key]
		if entry != nil && !entry.HasSetDef {
			calls := getCalls[key]
			for _, c := range calls {
				fmt.Printf("  - %s (viper.%s at %s:%d)\n", key, "Get"+c.Type, c.SourceFile, c.SourceLine)
			}
		}
	}
}
