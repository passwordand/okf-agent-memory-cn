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

function resolveProject(projectRoot, startCwd, globalBundle = GLOBAL_BUNDLE) {
  const found = projectRoot ? null : findBundle(startCwd);
  const root = realpath(path.resolve(projectRoot || (found ? path.dirname(found) : startCwd)));
  const target = path.join(root, "knowledge");
  let bundle = target;
  let bundleInfo;
  try {
    bundleInfo = fs.lstatSync(target);
  } catch (err) {
    if (err.code !== "ENOENT") throw err;
  }
  if (bundleInfo) {
    if (!bundleInfo.isDirectory() && !bundleInfo.isSymbolicLink()) {
      throw new Error(`PROJECT_BUNDLE_INVALID: ${target} is not a directory`);
    }
    try {
      bundle = realpath(target);
      if (!fs.statSync(bundle).isDirectory()) {
        throw new Error(`PROJECT_BUNDLE_INVALID: ${target} is not a directory`);
      }
    } catch (err) {
      if (err.code === "ENOENT") {
        throw new Error(`PROJECT_BUNDLE_INVALID: ${target} is a broken link`);
      }
      throw err;
    }
  }
  const relative = path.relative(root, bundle);
  if (relative === ".." || relative.startsWith(`..${path.sep}`) || path.isAbsolute(relative)) {
    throw new Error(`PROJECT_SCOPE_REJECTED_OUTSIDE_ROOT: ${bundle}`);
  }
  let globalReal;
  try {
    globalReal = realpath(globalBundle);
  } catch {
    globalReal = path.resolve(globalBundle);
  }
  if (samePath(bundle, globalReal)) {
    throw new Error(`PROJECT_SCOPE_REJECTED_GLOBAL: ${bundle}`);
  }
  return { bundle, cwd: bundleInfo ? bundle : root };
}

if (require.main === module) {
  const args = parseArgs(process.argv.slice(2));
  const startCwd = process.cwd();
  let resolved;
  try {
    if (args.scope === "global") {
      const globalBundle = resolveGlobal();
      resolved = { bundle: globalBundle, cwd: globalBundle };
    } else {
      resolved = resolveProject(args.projectRoot, startCwd);
    }
  } catch (err) {
    fail(2, err.message);
  }
  const { bundle, cwd } = resolved;

  if (!fs.existsSync(OKF)) fail(2, `OKF_NOT_FOUND: ${OKF}`);

  process.stderr.write(
    `okf-mcp scope=${args.scope} cwd=${startCwd} bundle=${bundle} root=${bundle}\n`,
  );

  const child = spawn(OKF, ["mcp", bundle], {
    stdio: "inherit",
    cwd,
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
}

module.exports = { resolveProject };
