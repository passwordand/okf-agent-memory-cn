const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { resolveProject } = require("./mem-mcp.js");

function tempRoot(t) {
  const tempDir = fs.realpathSync(os.tmpdir());
  const root = fs.mkdtempSync(path.join(tempDir, "okf-wrapper-"));
  t.after(() => {
    const target = fs.realpathSync(root);
    if (path.dirname(target) !== tempDir || !path.basename(target).startsWith("okf-wrapper-")) {
      throw new Error(`unsafe test cleanup path: ${target}`);
    }
    fs.rmSync(target, { recursive: true, force: true });
  });
  return root;
}

test("missing bundle uses the working directory without requiring Git", (t) => {
  const root = tempRoot(t);
  const result = resolveProject(null, root, path.join(root, "global"));
  assert.equal(result.bundle, path.join(root, "knowledge"));
  assert.equal(result.cwd, root);
  assert.equal(fs.existsSync(result.bundle), false);
});

test("explicit project root overrides the working directory", (t) => {
  const root = tempRoot(t);
  const project = path.join(root, "project");
  const nested = path.join(root, "other", "nested");
  fs.mkdirSync(project, { recursive: true });
  fs.mkdirSync(nested, { recursive: true });
  const result = resolveProject(project, nested, path.join(root, "global"));
  assert.equal(result.bundle, path.join(project, "knowledge"));
  assert.equal(result.cwd, project);
});

test("existing bundle keeps ancestor discovery and bundle working directory", (t) => {
  const root = tempRoot(t);
  const bundle = path.join(root, "knowledge");
  const nested = path.join(root, "src", "module");
  fs.mkdirSync(bundle);
  fs.mkdirSync(nested, { recursive: true });
  fs.writeFileSync(path.join(bundle, "index.md"), "---\nokf_version: \"0.2\"\n---\n");
  const result = resolveProject(null, nested, path.join(root, "global"));
  assert.equal(result.bundle, bundle);
  assert.equal(result.cwd, bundle);
});

test("project mode rejects the global bundle", (t) => {
  const root = tempRoot(t);
  const global = path.join(root, "knowledge");
  fs.mkdirSync(global);
  assert.throws(() => resolveProject(root, root, global), /PROJECT_SCOPE_REJECTED_GLOBAL/);
});

test("project mode rejects a symlink escaping the project root", (t) => {
  const root = tempRoot(t);
  const project = path.join(root, "project");
  const outside = path.join(root, "outside");
  fs.mkdirSync(project);
  fs.mkdirSync(outside);
  try {
    fs.symlinkSync(outside, path.join(project, "knowledge"), process.platform === "win32" ? "junction" : "dir");
  } catch (err) {
    if (err.code === "EPERM" || err.code === "EACCES") {
      t.skip("current system cannot create directory links");
      return;
    }
    throw err;
  }
  assert.throws(() => resolveProject(project, project, path.join(root, "global")), /PROJECT_SCOPE_REJECTED_OUTSIDE_ROOT/);
});

test("project mode rejects a broken knowledge link", (t) => {
  const root = tempRoot(t);
  const project = path.join(root, "project");
  fs.mkdirSync(project);
  try {
    fs.symlinkSync(path.join(root, "missing"), path.join(project, "knowledge"), process.platform === "win32" ? "junction" : "dir");
  } catch (err) {
    if (err.code === "EPERM" || err.code === "EACCES") {
      t.skip("current system cannot create directory links");
      return;
    }
    throw err;
  }
  assert.throws(() => resolveProject(project, project, path.join(root, "global")), /PROJECT_BUNDLE_INVALID/);
});
