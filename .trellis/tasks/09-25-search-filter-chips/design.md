# 技术设计：SearchFilterTemplate 已选条件 chips 回显行

## 1. 状态模型

chips 是「已提交筛选」的镜像，状态放在模板内部，消费方零改动：

```
handleFinish(values) ──► setCommitted(values) ──► onSearch(values)   （现有链路不变）
handleReset()        ──► setCommitted({})     ──► form.resetFields() ──► onReset()
deleteChip(name)     ──► setCommitted(去掉name) ──► form.setFieldValue(name, undefined)
                     ──► onSearch(去掉name后的完整对象)
```

- `committed: TValues` 由 `useState` 持有，仅在 `onFinish` / `onReset` / 删除 chip 时变更。
- 删除 chip 后必须同时 `form.setFieldValue(name, undefined)`：否则表单仍显示该值，下次点「查询」会把已删除的条件重新带回，语义矛盾。
- 已验证全部消费方的 `onSearch` 回调为**整体替换**语义（`setSearchParams(values)` / `setFilterParams(values)`，见 `finance/bills/index.tsx:437`、`finance/fees/index.tsx:144`），因此「剔除字段后的对象」传入后口径正确，无需逐页改造。

## 2. chip 展示值解析

新增纯函数（组件文件内即可，暂不独立导出）：

```
resolveChipText(item, value, labelMap) → string | null
```

| 字段类型 | 展示值来源 | 空值判定 |
|---|---|---|
| `input` / `digit` | `String(value).trim()` | 空串 → null |
| `select` / `searchable-select` | 静态：`item.options` 中按 value 匹配 label；远程：`labelMap[item.name]`（见 §3），无捕获则回退 `String(value)` | value === undefined/null/'' → null |
| `date` | `formatDate(value, 'date')`（`utils/format`，本地时区） | 同上 |
| `date-range` | `formatDate(v[0],'date') ~ formatDate(v[1],'date')` | 两端皆空 → null；单端有值时空端以 `-` 占位（RangePicker 交互恒为成对值，仅程序化初始值可触发） |
| `custom` | 不产生 chip（无法推断展示值） | — |

- 时间格式复用 `web/src/utils/format.ts` 的 `formatDate`，符合仓库时间显示规范（禁止 ISO 原文直出）。
- `keyword` 空串在 antd Form 中常见（allowClear 清空后 submit），必须过滤。

## 3. 远程下拉 label 捕获

`request` 型字段的候选项在 `RemoteSearchSelect` 内部异步加载，模板拿不到 options。方案：**在选择动作发生时捕获 label**。

- `renderFieldInput` 对 `select` / `searchable-select` 类型的 onChange 做合并拦截（不破坏消费方传入的 `fieldProps.onChange`）：
  ```ts
  const mergedOnChange = (value, option) => {
    labelMapRef.current[item.name] = extractLabel(option); // option 可为数组（multiple）
    fieldProps?.onChange?.(value, option);
  };
  ```
- `SearchableSelect` / `RemoteSearchSelect` 均把 rest 透传给 antd `Select`，onChange 的 `(value, option)` 签名可达；`extractLabel` 兼容单值与数组。
- 未捕获（如未来支持 initialValue 预填远程字段）时回退显示原值，不报错。
- `labelMapRef` 用 `useRef` 保持引用稳定（符合仓库「纯动作型回调引用稳定性」约束），重置/清空时同步清空对应项。

## 4. 渲染位置与视觉

- **grid 模式**：chips 行渲染在 `<Form>` 内、字段 `<Row>` 上方，仅当 `committed` 非空时出现；样式沿用模板 Card 白底规范，`Tag closable` 默认色 + 13px 字号，行尾 `<Button type="link">清除全部</Button>`。
- **bar 模式**：表单行下方追加第二行（同条件渲染），「清除全部」同上。
- **custom 模式**：不渲染（无消费方，children 结构无法推断字段）。
- chips 行的出现/消失不改变折叠逻辑（`visibleItems`、`actionSpan` 计算不变）。

## 5. 兼容性与风险

| 风险 | 评估 | 处置 |
|---|---|---|
| 消费方 onSearch 非整体替换（局部合并） | 已核对 bills/fees 为替换语义；commissions/partners/users/roles 走同一模板模式 | 实现阶段抽查其余消费方确认；若发现合并语义页单独调整该页 |
| 删除 chip 与进行中的远程请求竞态 | 与「重置」现有行为同级 | 不特殊处理，依赖消费方 request 防竞态（ProTable 内建 + React Query） |
| TValues 含 Dayjs 等复杂对象 | 只读展示，不做深拷贝 | 无 |
| chips 状态与外部 topBar 筛选（公司/标签）无关联 | 两者本就独立状态源 | 维持现状，文档说明 |

不新增任何 props：默认开启、无开关（无消费方需要关闭，YAGNI）；`layout="custom"` 靠模式判断天然跳过。

## 6. 测试策略

扩展现有 `SearchFilterTemplate.test.tsx`（渲染类用例已有 5 个），新增行为用例：

1. grid：填 input + select → 提交 → 断言 chips 文案与 label 解析；
2. grid：空串提交不产生 chip；
3. grid：删除 chip → 表单字段清空 + onSearch 参数不含该字段且含其余字段；
4. grid：date-range chip 本地格式文案 + 清除全部触发 onReset；
5. bar：quickFilters 提交后回显与删除；
6. 远程字段（mock request）：选择后提交，chip 显示所选 label。
