import { customPinyin, pinyin } from 'pinyin-pro';

// 与服务端 searchtext 的业务词组读音保持一致。
customPinyin({ 闵行: 'mǐn háng', 还箱: 'huán xiāng', 还柜: 'huán guì' });

// 有界缓存避免大列表每次按键重复转写，也不无限保留业务名称。
const searchKeys = new Map<string, string[]>();
const MAX_CACHED_TEXTS = 5000;

function getSearchKeys(text: string): string[] {
  const cached = searchKeys.get(text);
  if (cached) return cached;
  const syllables = pinyin(text, {
    toneType: 'none',
    type: 'array',
    nonZh: 'removed',
  });
  const keys = [
    syllables.join(''),
    syllables.join(' '),
    syllables.map((syllable) => syllable[0]).join(''),
  ];
  if (searchKeys.size >= MAX_CACHED_TEXTS) searchKeys.clear();
  searchKeys.set(text, keys);
  return keys;
}

/** 名称搜索统一覆盖原文、无声调全拼与首字母；不同字段分别匹配。 */
export function matchesSearch(keyword: string, ...values: unknown[]): boolean {
  const target = keyword.trim().toLowerCase();
  if (!target) return true;
  const compactTarget = target.replace(/\s+/g, '');
  return values.some((value) => {
    if (typeof value !== 'string' && typeof value !== 'number') return false;
    const text = String(value).toLowerCase();
    if (text.includes(target)) return true;
    if (!/\p{Script=Han}/u.test(text)) return false;
    return getSearchKeys(text).some(
      (key) => key.includes(target) || key.includes(compactTarget),
    );
  });
}
