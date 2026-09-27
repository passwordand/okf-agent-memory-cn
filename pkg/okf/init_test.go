package okf_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/okf-memory/okf-agent-memory/pkg/okf"
)

func TestInitBundlePreservesExistingFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "knowledge")
	if err := okf.InitBundle(dir); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(dir, "index.md")
	custom := "---\nokf_version: \"0.2\"\n---\n\n# 用户内容\n"
	if err := os.WriteFile(indexPath, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "log.md")); err != nil {
		t.Fatal(err)
	}
	if err := okf.InitBundle(dir); err != nil {
		t.Fatal(err)
	}
	if err := okf.InitBundle(dir); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(indexPath)
	if err != nil || string(index) != custom {
		t.Fatalf("重复初始化覆盖了现有 index.md: %v, %q", err, index)
	}
	if _, err := os.Stat(filepath.Join(dir, "log.md")); err != nil {
		t.Fatalf("缺失的 log.md 未补齐: %v", err)
	}
}

func TestInitBundleRejectsNonRegularFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "knowledge")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "index.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := okf.InitBundle(dir); err == nil {
		t.Fatal("index.md 为目录时应拒绝初始化")
	}
	if _, err := os.Stat(filepath.Join(dir, "log.md")); !os.IsNotExist(err) {
		t.Fatalf("预检查失败时不应写入 log.md: %v", err)
	}

	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked-knowledge")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("当前系统不能创建符号链接: %v", err)
	}
	if err := okf.InitBundle(link); err == nil {
		t.Fatal("知识库目录为符号链接时应拒绝初始化")
	}
	if _, err := os.Stat(filepath.Join(outside, "index.md")); !os.IsNotExist(err) {
		t.Fatalf("符号链接目标不应被写入: %v", err)
	}
}

func TestInitBundleReportsPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACL 不由 chmod 写入位控制")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(root, 0o755) }()
	err := okf.InitBundle(filepath.Join(root, "knowledge"))
	if err == nil {
		t.Skip("当前进程仍可写入只读目录")
	}
	if !strings.Contains(err.Error(), "failed to create directory") {
		t.Fatalf("权限错误未被明确报告: %v", err)
	}
}
