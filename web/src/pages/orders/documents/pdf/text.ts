import fontCoverage from './fontCoverage.json';
import fontGlyphWidths from './fontGlyphWidths.json';

export function hasPdfGlyph(character: string): boolean {
  if (/^[\n\r\t]$/.test(character)) return true;
  const point = character.codePointAt(0);
  if (point === undefined) return true;
  let start = 0;
  let end = fontCoverage.length - 1;
  while (start <= end) {
    const middle = Math.floor((start + end) / 2);
    const [first, last] = fontCoverage[middle];
    if (point < first) end = middle - 1;
    else if (point > last) start = middle + 1;
    else return true;
  }
  return false;
}

export function pdfText(value?: string | null, maxGlyphs?: number): string {
  const text = value?.replace(/\r\n?/g, '\n').replace(/\t/g, '    ') ?? '';
  if (!maxGlyphs) return text;
  // 按列宽显式拆行；ASCII 与超宽字形使用嵌入字体的 advance，
  // 其余字形按一字一 em 的保守上界，只换行，不插入连字符。
  return text
    .split('\n')
    .map((word) => {
      const lines: string[] = [];
      let line = '';
      let width = 0;
      for (const char of word) {
        const advance =
          fontGlyphWidths[char as keyof typeof fontGlyphWidths] ?? 1;
        if (line && width + advance > maxGlyphs) {
          lines.push(line);
          line = '';
          width = 0;
        }
        line += char;
        width += advance;
      }
      if (line) lines.push(line);
      return lines.join('\n');
    })
    .join('\n');
}

/** 正式单证使用英文；保留原值并让调用方提示字段，不删除或翻译自由正文。 */
export function hasNonEnglishPdfText(value: string): boolean {
  return [...value].some(
    (character) =>
      (/\p{Letter}/u.test(character) && !/\p{Script=Latin}/u.test(character)) ||
      /[\u3000-\u303f\uff00-\uffef]/u.test(character),
  );
}
