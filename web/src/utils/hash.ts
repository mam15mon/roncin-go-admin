/**
 * 确定性 Canonical JSON 序列化与 SHA-256 哈希工具
 * 用于生成稳定的请求指纹 (requestFingerprint) 与幂等键 (idempotencyKey)
 */

export function canonicalStringify(val: unknown): string {
  if (val === null || typeof val !== 'object') {
    return JSON.stringify(val);
  }
  if (Array.isArray(val)) {
    return `[${val.map(canonicalStringify).join(',')}]`;
  }
  const objectValue = val as Record<string, unknown>;
  const keys = Object.keys(objectValue).sort();
  const pairs: string[] = [];
  for (const k of keys) {
    if (objectValue[k] !== undefined) {
      pairs.push(`${JSON.stringify(k)}:${canonicalStringify(objectValue[k])}`);
    }
  }
  return `{${pairs.join(',')}}`;
}

function rightRotate(value: number, amount: number): number {
  return (value >>> amount) | (value << (32 - amount));
}

export function sha256(str: string): string {
  const ascii = unescape(encodeURIComponent(str));
  return sha256Bytes(
    Uint8Array.from(ascii, (character) => character.charCodeAt(0)),
  );
}

/** 对原始字节计算 SHA-256，支持没有 Web Crypto 的局域网 HTTP 页面。 */
export function sha256Bytes(bytes: Uint8Array): string {
  const mathPow = Math.pow;
  const maxWord = mathPow(2, 32);
  let result = '';
  const asciiBitLength = bytes.length * 8;
  const hash: number[] = [];
  const k: number[] = [];
  let primeCounter = 0;
  const isComposite: Record<number, number> = {};

  for (let candidate = 2; primeCounter < 64; candidate++) {
    if (!isComposite[candidate]) {
      for (let i = 0; i < 313; i += candidate) {
        isComposite[i] = candidate;
      }
      hash[primeCounter] = (mathPow(candidate, 0.5) * maxWord) | 0;
      k[primeCounter++] = (mathPow(candidate, 1 / 3) * maxWord) | 0;
    }
  }

  // 分块构造消息，避免把大附件再复制成整份填充字节和 number[]。
  const blockCount = Math.ceil((bytes.length + 9) / 64);
  for (let block = 0; block < blockCount; block++) {
    const w: number[] = new Array(16).fill(0);
    for (let offset = 0; offset < 64; offset++) {
      const position = block * 64 + offset;
      const byte =
        position < bytes.length
          ? bytes[position]
          : position === bytes.length
            ? 0x80
            : 0;
      w[offset >> 2] |= byte << ((3 - (offset % 4)) * 8);
    }
    if (block === blockCount - 1) {
      w[14] = (asciiBitLength / maxWord) | 0;
      w[15] = asciiBitLength;
    }
    const workingHash = hash.slice(0, 8);

    for (let i = 0; i < 64; i++) {
      const w15 = w[i - 15];
      const w2 = w[i - 2];
      const a = workingHash[0];
      const e = workingHash[4];

      const s1 = rightRotate(e, 6) ^ rightRotate(e, 11) ^ rightRotate(e, 25);
      const ch = (e & workingHash[5]) ^ (~e & workingHash[6]);
      if (i >= 16) {
        const s0 = rightRotate(w15, 7) ^ rightRotate(w15, 18) ^ (w15 >>> 3);
        const s1w = rightRotate(w2, 17) ^ rightRotate(w2, 19) ^ (w2 >>> 10);
        w[i] = (w[i - 16] + s0 + w[i - 7] + s1w) | 0;
      }

      const temp1 = workingHash[7] + s1 + ch + k[i] + w[i];
      const s0 = rightRotate(a, 2) ^ rightRotate(a, 13) ^ rightRotate(a, 22);
      const maj =
        (a & workingHash[1]) ^
        (a & workingHash[2]) ^
        (workingHash[1] & workingHash[2]);
      const temp2 = s0 + maj;

      workingHash.pop();
      workingHash.unshift((temp1 + temp2) | 0);
      workingHash[4] = (workingHash[4] + temp1) | 0;
    }

    for (let i = 0; i < 8; i++) {
      hash[i] = (hash[i] + workingHash[i]) | 0;
    }
  }

  for (let i = 0; i < 8; i++) {
    for (let j = 3; j >= 0; j--) {
      const b = (hash[i] >> (j * 8)) & 255;
      result += (b < 16 ? '0' : '') + b.toString(16);
    }
  }
  return result;
}

export function computeCanonicalSha256(payload: unknown): string {
  const canonical = canonicalStringify(payload);
  return sha256(canonical);
}
