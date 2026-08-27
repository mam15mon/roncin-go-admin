const baseURL = (
  process.env.RONCIN_ACCEPTANCE_BASE_URL || 'http://127.0.0.1:8000'
).replace(/\/$/, '');

function requireEnvironment(name) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`缺少环境变量 ${name}`);
  return value;
}

async function readJSON(response) {
  const text = await response.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    throw new Error(`${response.url} 返回了非 JSON 响应`);
  }
}

async function login() {
  const response = await fetch(`${baseURL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      username: requireEnvironment('BOOTSTRAP_ADMIN_USERNAME'),
      password: requireEnvironment('BOOTSTRAP_ADMIN_PASSWORD'),
    }),
  });
  const body = await readJSON(response);
  if (!response.ok) {
    throw new Error(
      `管理员登录失败（HTTP ${response.status}）：${body.message || '未知错误'}`,
    );
  }
  const cookie = response.headers
    .getSetCookie()
    .map((value) => value.split(';', 1)[0])
    .join('; ');
  if (!cookie) throw new Error('管理员登录成功但响应未设置会话 Cookie');
  return cookie;
}

function createClient(cookie) {
  return async function raw(path, options = {}) {
    const response = await fetch(`${baseURL}${path}`, {
      ...options,
      headers: {
        cookie,
        ...(options.body ? { 'content-type': 'application/json' } : {}),
        ...options.headers,
      },
    });
    return { response, body: await readJSON(response) };
  };
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function preferenceBody(preference, version) {
  return {
    columns: preference.columns || [],
    pageSize: preference.pageSize,
    sortField: preference.sortField,
    sortDirection: preference.sortDirection,
    rowColors: preference.rowColors,
    version,
  };
}

const cookie = await login();
const raw = createClient(cookie);
const initialResult = await raw('/api/v1/finance/fees/preference');
assert(initialResult.response.ok, '读取费用明细表头设置失败');
const original = initialResult.body.data;
const originalVersion = Number(original.version || 0);
assert(original?.pageSize === 40 || original?.customized, '系统默认分页行数应为 40');
assert(original?.rowColors?.completed, '系统默认列表颜色缺失');

const columns = Array.from({ length: 153 }, (_, index) => ({
  fieldKey: `field${String(index + 1).padStart(3, '0')}`,
  visible: index % 3 !== 0,
}));
const common = {
  columns,
  sortField: 'field153',
  sortDirection: 'DESC',
  rowColors: {
    unbilled: '#FFF7E6',
    unverifiedUninvoiced: '#FFFBE6',
    invoicedUnverified: '#E6F4FF',
    verifiedUninvoiced: '#F9F0FF',
    completed: '#F6FFED',
  },
  version: originalVersion,
};

const invalidResult = await raw('/api/v1/finance/fees/preference', {
  method: 'PUT',
  body: JSON.stringify({
    ...common,
    pageSize: 20,
  }),
});
assert(invalidResult.response.status === 400, '非法分页规格未被拒绝');

const [first, second] = await Promise.all([
  raw('/api/v1/finance/fees/preference', {
    method: 'PUT',
    body: JSON.stringify({ ...common, pageSize: 60 }),
  }),
  raw('/api/v1/finance/fees/preference', {
    method: 'PUT',
    body: JSON.stringify({ ...common, pageSize: 100 }),
  }),
]);
const winners = [first, second].filter((result) => result.response.ok);
const conflicts = [first, second].filter(
  (result) => result.response.status === 409,
);
assert(winners.length === 1 && conflicts.length === 1, '并发保存未实现单写入与版本冲突保护');
const saved = winners[0].body.data;
assert(
  saved.customized && Number(saved.version || 0) > originalVersion,
  '保存后版本号或自定义标记不正确',
);

const loadedResult = await raw('/api/v1/finance/fees/preference');
assert(loadedResult.response.ok, '保存后读取费用明细表头设置失败');
const loaded = loadedResult.body.data;
assert(loaded.columns?.length === 153, '后端未完整保存 153 项字段设置');
assert(
  loaded.columns[0].fieldKey === 'field001' &&
    loaded.columns[152].fieldKey === 'field153',
  '后端未保持字段排列顺序',
);
assert(
  loaded.pageSize === saved.pageSize &&
    loaded.sortField === 'field153' &&
    loaded.sortDirection === 'DESC',
  '分页或默认排序设置未正确回读',
);
assert(
  loaded.rowColors.completed === '#F6FFED',
  '列表颜色设置未正确回读',
);

let restored;
if (original.customized) {
  const restoreResult = await raw('/api/v1/finance/fees/preference', {
    method: 'PUT',
    body: JSON.stringify(preferenceBody(original, loaded.version)),
  });
  assert(restoreResult.response.ok, '恢复原费用明细表头设置失败');
  restored = restoreResult.body.data;
} else {
  const resetResult = await raw(
    `/api/v1/finance/fees/preference?version=${loaded.version}`,
    { method: 'DELETE' },
  );
  assert(resetResult.response.ok, '重置费用明细表头设置失败');
  restored = resetResult.body.data;
}
assert(restored.customized === original.customized, '验收后未恢复原自定义状态');

console.log('费用明细表头设置真实 API 验收通过。');
console.log(
  JSON.stringify(
    {
      fieldCount: loaded.columns.length,
      columnOrderVerified: true,
      pageSizeAndSortVerified: true,
      rowColorsVerified: true,
      concurrentVersionConflictVerified: true,
      originalPreferenceRestored: true,
    },
    null,
    2,
  ),
);
