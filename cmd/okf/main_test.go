package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInitDefaultCreatesKnowledgeBundle(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	cmdInit(nil)
	for _, name := range []string{"index.md", "log.md"} {
		if _, err := os.Stat(filepath.Join("knowledge", name)); err != nil {
			t.Fatalf("knowledge/%s 未创建: %v", name, err)
		}
	}
	if _, err := os.Stat("index.md"); !os.IsNotExist(err) {
		t.Fatalf("项目根目录不应出现 index.md，得到: %v", err)
	}
	customIndex := []byte("---\nokf_version: \"0.2\"\n---\n\n# 保留现有内容\n")
	if err := os.WriteFile(filepath.Join("knowledge", "index.md"), customIndex, 0o644); err != nil {
		t.Fatal(err)
	}
	cmdInit(nil)
	keptIndex, err := os.ReadFile(filepath.Join("knowledge", "index.md"))
	if err != nil || string(keptIndex) != string(customIndex) {
		t.Fatalf("重复初始化覆盖了现有文件: %v, %q", err, keptIndex)
	}

	cmdInit([]string{"."})
	if _, err := os.Stat("index.md"); err != nil {
		t.Fatalf("显式路径 . 未创建根目录知识库: %v", err)
	}
	cmdInit([]string{"custom"})
	if _, err := os.Stat(filepath.Join("custom", "index.md")); err != nil {
		t.Fatalf("自定义路径未创建知识库: %v", err)
	}
}

func TestDefaultBundlePrefersExistingRootAndNewKnowledge(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	if got, _ := defaultBundle(nil); got != "knowledge" {
		t.Fatalf("空项目默认 bundle = %q，期望 knowledge", got)
	}
	if err := os.WriteFile("index.md", []byte("---\nokf_version: \"0.2\"\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := defaultBundle(nil); got != "." {
		t.Fatalf("已有根目录 bundle 的默认路径 = %q，期望 .", got)
	}
	if err := os.Mkdir("knowledge", 0o755); err != nil {
		t.Fatal(err)
	}
	if got, _ := defaultBundle(nil); got != "knowledge" {
		t.Fatalf("项目 bundle 的默认路径 = %q，期望 knowledge", got)
	}
}

func TestSplitOptionalPath(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		fallback string
		wantPath string
		wantArgs []string
	}{
		{
			name:     "flags only",
			args:     []string{"--limit", "3", "--json"},
			fallback: "knowledge",
			wantPath: "knowledge",
			wantArgs: []string{"--limit", "3", "--json"},
		},
		{
			name:     "explicit path before flags",
			args:     []string{"custom-bundle", "--limit", "3"},
			fallback: "knowledge",
			wantPath: "custom-bundle",
			wantArgs: []string{"--limit", "3"},
		},
		{
			name:     "valued create flags",
			args:     []string{"--type", "Fact", "--title", "Demo", "--desc", "Text"},
			fallback: "knowledge",
			wantPath: "knowledge",
			wantArgs: []string{"--type", "Fact", "--title", "Demo", "--desc", "Text"},
		},
		{
			name:     "valued update flags",
			args:     []string{"--desc", "Updated text", "--actor", "agent/test"},
			fallback: "knowledge",
			wantPath: "knowledge",
			wantArgs: []string{"--desc", "Updated text", "--actor", "agent/test"},
		},
		{
			name:     "valued relate flags",
			args:     []string{"--desc", "Relationship context", "--actor", "agent/test"},
			fallback: "knowledge",
			wantPath: "knowledge",
			wantArgs: []string{"--desc", "Relationship context", "--actor", "agent/test"},
		},
		{
			name:     "valued bootstrap flags",
			args:     []string{"--name", "Demo", "--no-skill"},
			fallback: ".",
			wantPath: ".",
			wantArgs: []string{"--name", "Demo", "--no-skill"},
		},
		{
			name:     "no arguments",
			fallback: ".",
			wantPath: ".",
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotArgs := splitOptionalPath(tt.args, tt.fallback)
			if gotPath != tt.wantPath {
				t.Fatalf("splitOptionalPath() path = %q, want %q", gotPath, tt.wantPath)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Fatalf("splitOptionalPath() args = %#v, want %#v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestHasHelpFlag(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"--help"}, true},
		{[]string{"-h"}, true},
		{[]string{"help"}, true},
		{[]string{"architecture/database", "--help"}, true},
		{[]string{"architecture/database", "-h"}, true},
		{[]string{"--type", "Decision", "--help"}, true},
		{[]string{"architecture/database"}, false},
		{[]string{"--type", "Fact"}, false},
		{nil, false},
	}

	for _, tt := range tests {
		if got := hasHelpFlag(tt.args); got != tt.want {
			t.Errorf("hasHelpFlag(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}
