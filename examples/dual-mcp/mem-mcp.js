#!/usr/bin/env node
const { spawn } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");

const ROOT = path.join(os.homedir(), ".config", "agent-memory");
const GLOBAL_BUNDLE = path.join(ROOT, "knowledge");
const OKF = path.join(ROOT, "bin", process.platform === "win32" ? "okf.exe" : "okf");

function fail(code, message) {
  process.stderr.write(message + "\n");
  process.exit(code);
}

function parseArgs(argv) {
  const out = { scope: null, projectRoot: null };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--scope") {
      out.scope = argv[i + 1];
      i += 1;
      continue;
    }
    if (arg.startsWith("--scope=")) {
      out.scope = arg.slice("--scope=".length);
      continue;
    }
    if (arg === "--project-root") {
      out.projectRoot = argv[i + 1];
      i += 1;
      continue;
    }
    if (arg.startsWith("--project-root=")) {
      out.projectRoot = arg.slice("--project-root=".length);
      continue;
    }
    fail(2, `UNKNOWN_ARG: ${arg}`);
  }
  if (out.scope == null || out.scope === "") fail(2, "MISSING_SCOPE: require --scope project|global");
  if (out.scope !== "project" && out.scope !== "global") fail(2, `UNKNOWN_SCOPE: ${out.scope}`);
  if (out.scope === "global" && out.projectRoot) fail(2, "GLOBAL_SCOPE_HAS_PROJECT_ROOT");
  if (out.scope === "project" && out.projectRoot === "") fail(2, "EMPTY_PROJECT_ROOT");
  return out;
}

function realpath(p) {
  return fs.realpathSync.native ? fs.realpathSync.native(p) : fs.realpathSync(p);
}

function samePath(a, b) {
  const left = process.platform === "win32" ? a.toLowerCase() : a;
  const right = process.platform === "win32" ? b.toLowerCase() : b;
  return left === right;
}

function findBundle(start) {
  let dir = path.resolve(start);
  for (;;) {
    if (fs.existsSync(path.join(dir, "knowledge", "index.md"))) {
      return path.join(dir, "knowledge");
    }
    const parent = path.dirname(dir);
    if (parent === dir) return null;
    dir = parent;
  }
}

function resolveGlobal() {
  const index = path.join(GLOBAL_BUNDLE, "index.md");
  if (!fs.existsSync(index)) fail(2, `GLOBAL_BUNDLE_NOT_FOUND: ${GLOBAL_BUNDLE}`);
  return realpath(GLOBAL_BUNDLE);
}

function resolveProject(projectRoot, startCwd) {
  let bundle;
  let start = startCwd;
  if (projectRoot) {
    const root = path.resolve(projectRoot);
    start = root;
    if (!fs.existsSync(path.join(root, "knowledge", "index.md"))) {
      fail(2, `PROJECT_BUNDLE_NOT_FOUND: start=${root}`);
    }
    bundle = path.join(root, "knowledge");
  } else {
    bundle = findBundle(startCwd);
    if (!bundle) fail(2, `PROJECT_BUNDLE_NOT_FOUND: start=${startCwd}`);
  }
  const real = realpath(bundle);
  let globalReal;
  try {
    globalReal = realpath(GLOBAL_BUNDLE);
  } catch {
    globalReal = path.resolve(GLOBAL_BUNDLE);
  }
  if (samePath(real, globalReal)) {
    fail(2, `PROJECT_SCOPE_REJECTED_GLOBAL: ${real}`);
  }
  if (!fs.existsSync(path.join(real, "index.md"))) {
    fail(2, `PROJECT_BUNDLE_NOT_FOUND: start=${start}`);
  }
  return real;
}

const args = parseArgs(process.argv.slice(2));
const startCwd = process.cwd();
const bundle = args.scope === "global" ? resolveGlobal() : resolveProject(args.projectRoot, startCwd);

if (!fs.existsSync(OKF)) fail(2, `OKF_NOT_FOUND: ${OKF}`);

process.stderr.write(
  `okf-mcp scope=${args.scope} cwd=${startCwd} bundle=${bundle} root=${bundle}\n`,
);

const child = spawn(OKF, ["mcp", bundle], {
  stdio: "inherit",
  cwd: bundle,
  env: { ...process.env, OKF_MCP_ROOT: bundle },
  windowsHide: true,
});

function forward(signal) {
  if (child.pid && !child.killed) {
    try {
      child.kill(signal);
    } catch {
      child.kill();
    }
  }
}

process.on("SIGINT", () => forward("SIGINT"));
process.on("SIGTERM", () => forward("SIGTERM"));

child.on("error", (err) => fail(1, `OKF_SPAWN_FAILED: ${err.message}`));
child.on("exit", (code, signal) => {
  if (signal) process.exit(1);
  process.exit(code ?? 1);
});
