# 修复依据

- `web/scripts/duplicate-functions.mjs` 的 build/references 用 startsWith('TS') 跳过整个子树。`a(x:number){return x as number}` 与 `b(y:number){return x as number}` 被判为 renamed 相同，实际分别读取参数和自由变量；非空断言、satisfies 同样复现。
- `scripts/report-duplicates.mjs` 对源码前4096字符正则匹配生成标记；手写 `export function show(){return "@generated"}` 被排除且无错误。应识别文件头注释，不匹配任意业务字符串。
- 基线只校验 groups，阈值1产生1组、阈值60产生0组，源码未变却报告消失；生成命令只保存groups，必须同步升级元数据。
- 基线生成内联命令忽略 scan.errors，可能将失败扫描写成新基线。相邻路径需失败后保持旧基线不变。

## 验证边界
只验证工具、基线和扫描链路；不改产品页面。开始时存在 workbench 用户未提交改动，整个过程不重置、暂存或提交这些文件。
