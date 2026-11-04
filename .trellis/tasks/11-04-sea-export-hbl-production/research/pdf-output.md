# PDF 输出选型与字体验证

## 约束

要求同一份 HBL 数据生成可选取文本的 PDF，中英文正确、长内容可续页，预览/打印/下载一致。不引入服务端浏览器进程或外部转换平台。

## 官方依据

- [React-pdf 字体](https://react-pdf.org/docs/v4/fonts)：支持 TTF/WOFF，明确提示不直接使用 OpenType 可变字体。采用构建资产阶段静态化的 TTF，而非运行时加载可变字体。
- [React-pdf 分页](https://react-pdf.org/docs/v4/advanced/page-wrapping)：Text/View 可分页，fixed 可复用页眉页脚；超高内容不能使用不可分页块。
- [usePDF](https://react-pdf.org/docs/v4/hooks)：生成结果含 Blob URL、加载和错误；文档变化需显式更新，适合一份结果供多个入口消费。
- [PDF 查看器](https://react-pdf.org/docs/v4/components/pdf-viewer)：浏览器 iframe 预览，原生工具栏由浏览器支持。
- [PDF-LIB](https://pdf-lib.js.org/) 和 [fontkit](https://github.com/Hopding/fontkit)：浏览器 PDF 与自定义字体能力可用，但本轮实际字体子集试验出现缺字，不作为首期主渲染器。
- [Noto CJK 官方字体](https://github.com/notofonts/noto-cjk) 和 [OFL 许可证](https://raw.githubusercontent.com/notofonts/noto-cjk/main/Sans/LICENSE)：资产记录来源和授权，运行时从本站同域读取。

以上资料在 2026-11-04 核对。选择是基于本任务的版式与实测结果，不代表其他 PDF 工具均不能处理中文。

## 探针步骤与观察

所有试验位于临时目录，没有修改业务源码、仓库依赖或生产数据库。

1. 用 `pnpm` 在临时目录安装 `pdf-lib@1.17.1`、`@pdf-lib/fontkit@1.1.1`。
2. 使用官方 Noto CJK 常规 OTF 子集生成：文字可提取，但实际渲染报字体格式错误、中文未显示。
3. 下载官方 `Sans/Variable/TTF/Subset/NotoSansSC-VF.ttf`，用 FontTools `instantiateVariableFont` 将全部轴固定（wght=400），输出静态 TTF。
4. 同一静态 TTF 在 pdf-lib 的 `subset: true` 路径仍有部分字形不可见；完整嵌入虽可尝试，但简单文件已约 6.45 MB，不符合本期优先方案。
5. 使用 `@react-pdf/renderer@4.9.0` + 静态 TTF 生成同样的中英文、箱号、数值、四种版本标识探针。
6. 由独立 PDF 引擎提取文字并渲染为图片，目视确认所有中英文、数值均显示完整；探针 PDF 约 9.3 KB。此结果只验证字体/基础文本，不等同于 HBL 整页、浏览器和长内容验收。

探针运行时实际解析的 React 为 19.3.1，仓库当前安装 React 为 19.2.8；实施须用仓库锁定版本验证兼容，不为引入 PDF 强行升级 React。

## 字体溯源

- 核对时 Noto CJK 上游 HEAD：`f8d157532fbfaeda587e826d4cd5b21a49186f7c`。
- 源可变 TTF 大小：17,773,132 字节；SHA-256：`d68bafcb48a2707749396aa12bbbd833cb70401f3a9a689fd2902c7e0d295964`。
- 探针静态 TTF 大小：10,596,232 字节。实施时保存精确的源提交、轴设置、工具版本与最终资产校验和；字体编译时间戳可能影响二进制结果，不把本轮临时文件的哈希当通用构建常量。
- 静态化资产须随包包含 OFL 与来源说明；命名遵守字体授权，业务人员无需安装字体。

## 实施风险控制

- PDF 模块只在制作/预览路径懒加载，字体不进入普通订单首屏。
- 不在输入每个字符时生成整份 PDF；用户保存/预览或切换输出版本时生成。
- 长内容采用引擎正常分页，逐字段检查完整性，禁止靠隐藏溢出或缩成不可读的小字通过验收。
- 如果复杂表格存在分页缺陷，先修正模板布局；未经范围确认不改成服务器渲染或另一条静默输出路径。
