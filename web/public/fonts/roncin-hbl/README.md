# HBL PDF 中文字体

本目录字体仅供 HBL PDF 按需加载，普通订单页面不会请求此字体。浏览器从本站同域读取，PDF 只嵌入实际使用的字形，不依赖外部字体服务。

## 来源与授权

- 上游：[Noto CJK](https://github.com/notofonts/noto-cjk)。
- 固定源提交：`f8d157532fbfaeda587e826d4cd5b21a49186f7c`。
- 原文件：`Sans/Variable/TTF/Subset/NotoSansSC-VF.ttf`。
- 原文件 SHA-256：`d68bafcb48a2707749396aa12bbbd833cb70401f3a9a689fd2902c7e0d295964`。
- 许可证：[SIL Open Font License 1.1](./OFL.txt)，保留字体原有版权信息；静态衍生字体改名为 `Roncin HBL Sans SC`。
- 处理工具：Python FontTools `4.68.0`；固定 `wght=400`，无其他可变轴。
- 输出文件：`RoncinHBLSansSC-Regular.ttf`。
- 输出 SHA-256：`77ecc1242b6dfb247c483a7e94978197dbf0a7624d9be21cc0af79d4e328194a`。

## 更新方法

在独立工具环境中按固定源下载字体并验证 SHA-256。用 FontTools `varLib.instancer.instantiateVariableFont` 将所有轴固定，`wght` 为 400、其他轴为其默认值，再将 name 表中以下名称改为：

| name ID | 字段值 |
| --- | --- |
| 1 / 16 | Roncin HBL Sans SC |
| 2 / 17 | Regular |
| 3 / 4 | Roncin HBL Sans SC Regular |
| 6 | RoncinHBLSansSC-Regular |

保存为本目录的静态 TTF。不得直接将可变字体交给 PDF 渲染器。

同时从最终字体的 `TTFont(...).getBestCmap()` 读取所有 Unicode 码点，排序并合并成连续的 `[起点, 终点]` 区间，写入 `web/src/pages/orders/hbl/pdf/fontCoverage.json`。当前包含 30,890 个码点、448 个区间。该字符表用于生成前提示缺字字段，不能独立于字体更新。

从 `hmtx` 表读取各码点字形的 advance，除以 `head.unitsPerEm`；将 ASCII 可打印字符（U+0020～U+007E）以及宽度大于 1 em 的字形写入同目录 `fontGlyphWidths.json`。当前包含 95 个 ASCII 字符与 3 个超宽字形（`Ю`、`⸺`、`⸻`）。其余字形按 1 em 的保守上界折行，避免无空格编号和中文越过单元格；原始字符保持完整，不添加连字符。

更新后重新记录二进制校验和与工具版本，并实际渲染中英文、罕见字、长内容和四种版本。只检查提取文本或 PDF 文件存在不能证明字形正确显示。字体不支持的字符应明确报错，不得静默输出空白。
