import assert from 'node:assert/strict';

export function requiredEnvironment(name) {
  const value = process.env[name]?.trim();
  assert(value, `缺少环境变量 ${name}`);
  return value;
}

// Cookie只保留在进程内，不写证据或输出；请求失败只报告方法、路径和业务错误。
export async function createHgjClient(username, password) {
  const base = requiredEnvironment('RONCIN_ACCEPTANCE_BASE_URL').replace(/\/$/, '');
  let cookie = '';
  async function raw(path, options = {}) {
    const response = await fetch(`${base}${path}`, {
      ...options,
      headers: { ...(cookie ? { cookie } : {}), ...(options.body ? { 'content-type': 'application/json' } : {}), ...options.headers },
    });
    const rotation = response.headers.getSetCookie().map((item) => item.split(';', 1)[0]).join('; ');
    if (rotation) cookie = rotation;
    const text = await response.text();
    const body = text ? JSON.parse(text) : {};
    return { response, body };
  }
  async function request(path, options = {}) {
    const { response, body } = await raw(path, options);
    assert(response.ok && body.success === true, `${options.method || 'GET'} ${path} HTTP ${response.status}: ${body.message || body.reason || '请求失败'}`);
    return body;
  }
  const post = (path, data) => request(path, { method: 'POST', body: JSON.stringify(data) });
  const put = (path, data) => request(path, { method: 'PUT', body: JSON.stringify(data) });
  await post('/api/v1/auth/login', { username, password });
  return { request, raw, post, put };
}

export function masterNumber(number) {
  const serial = String(number).slice(-7).padStart(7, '0');
  return `990${serial}${Number(serial) % 7}`;
}

export const orderKinds = [
  { code: 'SE', type: 1, direction: 1, path: 'sea-export', air: false },
  { code: 'SI', type: 2, direction: 2, path: 'sea-import', air: false },
  { code: 'AE', type: 3, direction: 1, path: 'air-export', air: true },
  { code: 'AI', type: 4, direction: 2, path: 'air-import', air: true },
];
