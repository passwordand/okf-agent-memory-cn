// 用临时 HOME、真实 wrapper 和 stdio MCP 进程验证升级；不连接实际知识库或 Hub。
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');

const binary = path.resolve(process.argv[2]);
const legacy = path.resolve(process.argv[3]);
const output = path.resolve(process.argv[4]);
const expectedVersion = process.argv[5] || 'v0.6.1-cn.1';
const workspace = path.resolve(__dirname, '..');
const fixture = fs.mkdtempSync(path.join(fs.realpathSync.native(os.tmpdir()), 'okf-v061-isolation-'));
const home = path.join(fixture, 'home');
const project = path.join(fixture, 'project');
const globalBundle = path.join(home, '.config', 'agent-memory', 'knowledge');
const systemBundle = path.join(fixture, 'system');
const report = { fixture, binary, legacy, checks: [] };
const env = { ...process.env, USERPROFILE: home, HOME: home, OKF_USER_DIR: globalBundle, OKF_SYSTEM_DIR: systemBundle };
delete env.OKF_MCP_ROOT;
fs.mkdirSync(project, { recursive: true });
fs.mkdirSync(path.join(home, '.config', 'agent-memory', 'bin'), { recursive: true });
const installedBinary = path.join(home, '.config', 'agent-memory', 'bin', process.platform === 'win32' ? 'okf.exe' : 'okf');
fs.copyFileSync(binary, installedBinary);
if (process.platform !== 'win32') fs.chmodSync(installedBinary, 0o755);

function check(name, fn) {
  try { fn(); report.checks.push({ name, passed: true }); }
  catch (error) { report.checks.push({ name, passed: false, error: error.stack }); throw error; }
}
function cli(exe, args, cwd = project) {
  const result = spawnSync(exe, args, { env, cwd, encoding: 'utf8', timeout: 60000, windowsHide: true });
  assert.equal(result.status, 0, `${args.join(' ')}: ${result.stderr || result.error}`);
  return result.stdout;
}
function call(name, args = {}) { return { method: 'tools/call', params: { name, arguments: args } }; }
function session(scope, requests) {
  const input = requests.map((request, index) => JSON.stringify({ jsonrpc: '2.0', id: index + 1, ...request })).join('\n') + '\n';
  const result = spawnSync(process.execPath, [path.join(workspace, 'examples', 'dual-mcp', 'mem-mcp.js'), '--scope', scope], {
    env, cwd: project, input, encoding: 'utf8', timeout: 60000, windowsHide: true,
  });
  assert.equal(result.status, 0, result.stderr || String(result.error));
  const rows = result.stdout.trim().split(/\r?\n/).map(line => JSON.parse(line));
  assert.equal(rows.length, requests.length, 'MCP 响应数量不一致');
  return rows.map(row => { assert.equal(row.error, undefined); return row.result; });
}
function success(result) { assert.notEqual(result.isError, true, JSON.stringify(result)); return result.structuredContent; }
function rejected(result) { assert.equal(result.isError, true, JSON.stringify(result)); }

try {
  check('测试用 HOME 生效，避免接触真实全局库', () => {
    const probe = spawnSync(process.execPath, ['-p', 'require("os").homedir()'], { env, encoding: 'utf8', windowsHide: true });
    assert.equal(path.resolve(probe.stdout.trim()), home);
  });
  check('旧版创建全局库与中文概念，新版可读取', () => {
    cli(legacy, ['init', globalBundle]);
    cli(legacy, ['create', 'facts/shared', globalBundle, '--type', 'Fact', '--title', '全局约定', '--desc', '登录鉴权隔离标记']);
    assert.equal(JSON.parse(cli(binary, ['show', 'facts/shared', globalBundle, '--json'])).title, '全局约定');
    cli(binary, ['init', systemBundle]);
    cli(binary, ['create', 'facts/shared', systemBundle, '--type', 'Fact', '--title', '系统约定', '--desc', '登录鉴权隔离标记']);
  });
  check('缺项目库可握手，搜索报错，不回退全局、不自动建库', () => {
    const rows = session('project', [{ method: 'initialize', params: {} }, { method: 'tools/list' }, call('okf_search', { query: '登录鉴权' })]);
    assert.equal(rows[0].serverInfo.version, expectedVersion);
    assert.equal(rows[1].tools.length, 7);
    assert.deepEqual(rows[1].tools.find(tool => tool.name === 'okf_search').inputSchema.properties.scope.enum, ['bundle', 'project']);
    rejected(rows[2]);
    assert.equal(fs.existsSync(path.join(project, 'knowledge')), false);
  });
  check('同一 MCP 会话显式初始化、中文写入、元数据修改、过滤与重复初始化', () => {
    const rows = session('project', [
      call('okf_init'),
      call('okf_create', { concept_id: 'facts/shared', type: 'Fact', title: '项目约定', description: '登录鉴权隔离标记', tags: ['中文检索'], status: 'draft' }),
      call('okf_update', { concept_id: 'facts/shared', type: 'Decision', status: 'stable', tags: ['中文检索', '鉴权'] }),
      call('okf_show', { concept_id: 'facts/shared' }),
      call('okf_search', { query: '鉴权', filter: 'type=Decision,tags=中文检索' }),
      call('okf_validate', { strict: true }),
      call('okf_init'),
    ]);
    rows.forEach(success);
    assert.equal(rows[0].structuredContent.status, 'created');
    assert.equal(rows[3].structuredContent.type, 'Decision');
    assert.equal(rows[3].structuredContent.status, 'stable');
    assert.deepEqual(rows[3].structuredContent.tags, ['中文检索', '鉴权']);
    assert.equal(rows[4].structuredContent.results[0].title, '项目约定');
    assert.equal(rows[5].structuredContent.errors.length, 0);
    assert.equal(rows[5].structuredContent.warnings.length, 0);
    assert.equal(rows[6].structuredContent.status, 'already_initialized');
  });
  for (const [scope, title] of [['project', '项目约定'], ['global', '全局约定']]) {
    check(`${scope} 独立服务默认检索仅返回本库`, () => {
      const rows = session(scope, [call('okf_search', { query: '登录鉴权' }), call('okf_show', { concept_id: 'facts/shared' })]);
      const data = success(rows[0]);
      assert.equal(data.results.length, 1);
      assert.equal(data.results[0].title, title);
      assert.equal(success(rows[1]).title, title);
    });
    check(`${scope} 拒绝跨层搜索、URI 与越界 bundle 读写`, () => {
      const outside = scope === 'project' ? globalBundle : path.join(project, 'knowledge');
      const requests = ['all', 'user', 'system', 'vendor'].map(layer => call('okf_search', { query: '登录鉴权', scope: layer }));
      requests.push(...['user:facts/shared', 'system:facts/shared', 'okf://user/facts/shared', '@external/facts/shared'].map(id => call('okf_show', { concept_id: id })));
      requests.push(call('okf_show', { concept_id: 'facts/shared', bundle: outside }));
      requests.push(call('okf_update', { concept_id: 'facts/shared', bundle: outside, title: '不应写入' }));
      requests.push(call('okf_init', { bundle: outside }));
      session(scope, requests).forEach(rejected);
    });
  }
  check('新版写入后旧版仍可读取核心字段和中文检索', () => {
    const bundle = path.join(project, 'knowledge');
    const concept = JSON.parse(cli(legacy, ['show', 'facts/shared', bundle, '--json']));
    assert.equal(concept.title, '项目约定');
    assert.equal(concept.type, 'Decision');
    assert.deepEqual(concept.tags, ['中文检索', '鉴权']);
    const results = JSON.parse(cli(legacy, ['search', '鉴权', bundle, '--json']));
    assert.equal(results[0].concept_id, 'facts/shared');
    cli(legacy, ['validate', bundle, '--strict', '--drift']);
  });
  check('列表字段过滤与更新后仍为数组', () => {
    const bundle = path.join(project, 'knowledge');
    cli(binary, ['create', 'facts/list', bundle, '--type', 'Fact', '--title', '列表测试', '--desc', '列表字段回归']);
    const file = path.join(bundle, 'facts', 'list.md');
    fs.writeFileSync(file, fs.readFileSync(file, 'utf8').replace('---\n', '---\ntopics: [retrieval, memory]\n'));
    cli(binary, ['update', 'facts/list', bundle, '--desc', '更新列表字段回归']);
    const shown = JSON.parse(cli(binary, ['show', 'facts/list', bundle, '--json']));
    assert.deepEqual(shown.extra.topics, ['retrieval', 'memory']);
    const results = JSON.parse(cli(binary, ['search', '列表', bundle, '--filter', 'topics=retrieval', '--json']));
    assert.equal(results[0].concept_id, 'facts/list');
  });
  check('空标签清除保持可用，全局库内容未受项目操作影响', () => {
    const rows = session('project', [call('okf_update', { concept_id: 'facts/shared', tags: [] }), call('okf_show', { concept_id: 'facts/shared' })]);
    rows.forEach(success);
    assert.equal((rows[1].structuredContent.tags || []).length, 0);
    assert.equal(JSON.parse(cli(legacy, ['show', 'facts/shared', globalBundle, '--json'])).title, '全局约定');
  });
  check('Windows 目录 junction 不能将项目库指向外部库', () => {
    const other = path.join(fixture, 'junction-project');
    fs.mkdirSync(other);
    fs.symlinkSync(globalBundle, path.join(other, 'knowledge'), process.platform === 'win32' ? 'junction' : 'dir');
    const result = spawnSync(process.execPath, [path.join(workspace, 'examples', 'dual-mcp', 'mem-mcp.js'), '--scope', 'project', '--project-root', other], { env, cwd: other, encoding: 'utf8', timeout: 10000, windowsHide: true });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /PROJECT_SCOPE_REJECTED_OUTSIDE_ROOT/);
  });
  check('MCP 拒绝通过库内 junction 切换到外部 bundle', () => {
    const alias = path.join(project, 'knowledge', 'outside-link');
    fs.symlinkSync(globalBundle, alias, process.platform === 'win32' ? 'junction' : 'dir');
    const requests = ['okf_search', 'okf_show', 'okf_update'].map(name => call(name, {
      bundle: alias, query: '登录鉴权', concept_id: 'facts/shared', title: '不应写入',
    }));
    session('project', requests).forEach(rejected);
    session('project', [call('okf_create', {
      concept_id: 'outside-link/facts/escape', type: 'Fact', title: '不应写入', description: '越界创建检查',
    })]).forEach(rejected);
    const create = spawnSync(binary, ['create', 'outside-link/facts/escape', path.join(project, 'knowledge'), '--type', 'Fact', '--title', '不应写入', '--desc', '越界创建检查'], {
      env, cwd: project, encoding: 'utf8', timeout: 10000, windowsHide: true,
    });
    assert.notEqual(create.status, 0, create.stdout);
    assert.equal(fs.existsSync(path.join(globalBundle, 'facts', 'escape.md')), false);
    assert.equal(JSON.parse(cli(legacy, ['show', 'facts/shared', globalBundle, '--json'])).title, '全局约定');
  });
} finally {
  report.passed = report.checks.every(item => item.passed);
  fs.writeFileSync(output, JSON.stringify(report, null, 2) + '\n');
  console.log(JSON.stringify({ passed: report.passed, checks: report.checks.length, output }));
  // 仅清理本次新建且已校验的临时目录。
  const resolved = fs.realpathSync(fixture);
  if (path.dirname(resolved) === fs.realpathSync.native(os.tmpdir()) && path.basename(resolved).startsWith('okf-v061-isolation-')) {
    fs.rmSync(resolved, { recursive: true, force: true });
  }
}
