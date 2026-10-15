package okf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMCPBoundBundleToolSchemas(t *testing.T) {
	bound := &mcpServer{bundleOnly: true}
	for _, tool := range bound.toolDefinitions() {
		if tool["name"] == "okf_search" {
			props := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
			if got := props["scope"].(map[string]any)["enum"]; !reflect.DeepEqual(got, []string{"bundle", "project"}) {
				t.Fatalf("隔离服务暴露了跨层 scope: %+v", got)
			}
		}
	}
	for _, tool := range GetMCPTools() {
		if tool["name"] == "okf_search" {
			props := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
			if len(props["scope"].(map[string]any)["enum"].([]any)) != 6 {
				t.Fatal("隔离服务修改了共享工具定义")
			}
		}
	}
}

// 使用真实临时知识库，验证显式 MCP 根目录不会被多层读取绕过。
func TestMCPBoundBundleIsolation(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	global := filepath.Join(root, "global")
	system := filepath.Join(root, "system")
	vendor := filepath.Join(project, ".okf", "vendor", "outside")
	for _, dir := range []string{project, global, system, vendor} {
		if err := InitBundle(dir); err != nil {
			t.Fatal(err)
		}
		c := &Concept{ID: "facts/shared", Path: "facts/shared.md", Type: "Fact", Title: filepath.Base(dir), Description: "隔离检索标记", Status: "stable"}
		if err := SaveConcept(dir, c, SaveOptions{IsNew: true, AutoIndex: true, AutoLog: true, Actor: "agent/test"}); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("OKF_USER_DIR", global)
	t.Setenv("OKF_SYSTEM_DIR", system)
	call := func(bundle, name string, args map[string]any) map[string]any {
		t.Helper()
		request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
		if err != nil {
			t.Fatal(err)
		}
		responses := runMCPConversation(t, bundle, []string{string(request)})
		if len(responses) != 1 {
			t.Fatalf("响应数 = %d", len(responses))
		}
		result, ok := responses[0].Result.(map[string]any)
		if !ok {
			t.Fatalf("无效响应: %+v", responses[0])
		}
		return result
	}
	for _, bundle := range []string{project, global} {
		t.Run(filepath.Base(bundle), func(t *testing.T) {
			t.Setenv("OKF_MCP_ROOT", bundle)
			for _, scope := range []string{"", "bundle", "project"} {
				result := call(bundle, "okf_search", map[string]any{"query": "隔离检索", "scope": scope})
				if result["isError"] == true {
					t.Fatalf("本库检索失败: %+v", result)
				}
				data := result["structuredContent"].(map[string]any)
				items := data["results"].([]any)
				if len(items) != 1 || items[0].(map[string]any)["title"] != filepath.Base(bundle) {
					t.Fatalf("发生跨库混入: %+v", data)
				}
			}
			for _, scope := range []string{"all", "user", "system", "vendor", " ALL "} {
				if result := call(bundle, "okf_search", map[string]any{"query": "隔离检索", "scope": scope}); result["isError"] != true {
					t.Fatalf("跨层 scope %q 未被拒绝: %+v", scope, result)
				}
			}
			for _, id := range []string{"user:facts/shared", "system:facts/shared", "@outside/facts/shared", "okf://user/facts/shared", "okf://system/facts/shared", "okf://@outside/facts/shared", "../global/facts/shared", filepath.Join(global, "facts", "shared.md")} {
				if result := call(bundle, "okf_show", map[string]any{"concept_id": id}); result["isError"] != true {
					t.Fatalf("跨层引用 %q 未被拒绝: %+v", id, result)
				}
			}
			for _, id := range []string{"facts/shared", "facts/shared.md"} {
				result := call(bundle, "okf_show", map[string]any{"concept_id": id})
				if result["isError"] == true || result["structuredContent"].(map[string]any)["title"] != filepath.Base(bundle) {
					t.Fatalf("本库读取失败: %+v", result)
				}
			}
			outside := global
			if bundle == global {
				outside = project
			}
			for _, tool := range []string{"okf_search", "okf_show", "okf_create", "okf_update", "okf_relate", "okf_validate"} {
				result := call(bundle, tool, map[string]any{"bundle": outside, "query": "隔离检索", "concept_id": "facts/shared", "type": "Fact", "title": "不应写入"})
				if result["isError"] != true {
					t.Fatalf("%s 越界未被拒绝: %+v", tool, result)
				}
			}
		})
	}
	for _, dir := range []string{project, global, system, vendor} {
		b, err := LoadBundle(dir)
		if err != nil || b.Concepts["facts/shared"].Title != filepath.Base(dir) {
			t.Fatalf("越界请求修改了 %s: %v", dir, err)
		}
	}
	// 缺少项目库时，即使全局库包含同名数据，也不能静默返回全局结果。
	missing := filepath.Join(root, "missing")
	t.Setenv("OKF_MCP_ROOT", missing)
	for _, tool := range []string{"okf_search", "okf_show"} {
		result := call(missing, tool, map[string]any{"query": "隔离检索", "concept_id": "user:facts/shared"})
		if result["isError"] != true {
			t.Fatalf("缺库时静默回退: %+v", result)
		}
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("缺库查询创建了目录: %v", err)
	}
}

func TestChineseSearchWithV061Scoring(t *testing.T) {
	b := &Bundle{Concepts: map[string]*Concept{
		"facts/auth":  {ID: "facts/auth", Type: "Fact", Title: "登录鉴权", Description: "用户登录权限校验与 API token", Tags: []string{"中文检索", "鉴权"}},
		"facts/ui":    {ID: "facts/ui", Type: "Fact", Title: "UI 布局", Description: "界面排版"},
		"facts/other": {ID: "facts/other", Type: "Fact", Title: "Decision required", Description: "无关概念"},
	}}
	for query, want := range map[string]string{"登录": "facts/auth", "鉴权": "facts/auth", "API token": "facts/auth", "UI": "facts/ui"} {
		results, err := b.SearchAdvanced(SearchOptions{Query: query, Limit: 10})
		if err != nil || len(results) == 0 || results[0].ConceptID != want {
			t.Fatalf("查询 %q 召回失败: %+v, %v", query, results, err)
		}
		if strings.EqualFold(query, "UI") && len(results) != 1 {
			t.Fatalf("短词产生误匹配: %+v", results)
		}
	}
	results, err := b.SearchAdvanced(SearchOptions{Query: "鉴权", Filter: "tags=中文检索"})
	if err != nil || len(results) != 1 {
		t.Fatalf("中文标签过滤失败: %+v, %v", results, err)
	}
}
