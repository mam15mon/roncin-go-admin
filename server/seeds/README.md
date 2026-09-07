# 行业参考主数据（Industry Reference Seeds）

本目录用于归档海运与国际物流核心领域的官方基准数据（行业字典与种子数据）。
所有数据均来源于国际权威标准组织，用于在系统初始化或日常运维中一键落库，避免依赖外部不可靠接口或临时文件。

---

## 📁 文件清单与官方来源

| 文件名 | 数据领域 | 官方标准组织 | 数据版本 | 官方发布/下载页面 |
|---|---|---|---|---|
| **`loc251csv.zip`** | 全球港口五字码 (UN/LOCODE) | 联合国欧洲经济委员会 (UNECE) | 2025-1 | [UNECE UN/LOCODE Code List](https://unece.org/trade/cefact/unlocode-code-list-country-and-territory) |
| **`SMDG_Liner-codes-list-20260903.xlsx`** | 全球班轮船公司代码表 | 国际船舶规划系统用户协会 (SMDG) | 2026-09-03 | [SMDG Liner Code List](https://smdg.org/documents/smdg-liner-code-list/) |
| **`SMDG-Terminal-Code-List-v20260609.xlsx`** | 全球集装箱码头与泊位代码表 | 国际船舶规划系统用户协会 (SMDG) | 2026-06-09 | [SMDG Terminal Code List](https://smdg.org/documents/smdg-terminal-code-list/) |
| **`airlines.dat`** | 全球航空公司主数据 | OpenFlights 数据集 + 官方字典增强 | OpenFlights | [OpenFlights Airline Database](https://openflights.org/data.html) |

---

## 🚀 快速使用指南

项目根目录已集成开箱即用的自动化同步命令，无需手动配置参数即可直接读取本目录下的文件并落库。

### 1. 一键正式落库写入

在项目根目录下执行：

```bash
# 同步全球船公司主数据（自动导入 358+ 家船司、航运联盟分类、BIC 箱号前缀与中文对照）
pnpm run sync:shipping-lines

# 同步联合国港口数据（自动导入 17,524 个全球有效海港）
pnpm run sync:unlocode

# 同步全球航空公司主数据（自动导入 988 家活跃航司，136+ 家主流客货运航司中文名与 AWB 前缀）
pnpm run sync:airlines

# 同步全球机场主数据（自动从 OurAirports 拉取或使用本地 airports.csv）
pnpm run sync:airports
```

> **提示**：同步过程基于数据库唯一索引（如组织 + 港口五字码、组织 + 船司主代码、组织 + IATA/ICAO），多次重复执行具有**幂等性**（会自动执行增量更新，不重复新增）。

### 2. 预览模式（安全预检，不写入数据库）

如果想在落库前查看数据统计、有效性校验或冲突项，可运行预览命令：

```bash
# 预览船公司解析统计
go -C server run ./cmd/sync-shipping-lines

# 预览海港数据解析统计
go -C server run ./cmd/sync-unlocode

# 预览航司数据解析统计
go -C server run ./cmd/sync-airlines

# 预览机场数据解析统计
go -C server run ./cmd/sync-airports
```

输出示例：
```text
航司数据源：seeds/airlines.dat
组织：HQ，版本：OpenFlights，SHA-256：39be1a43...
原始行 6162，有效活跃航司 988（中文对照 136，运单前缀 136，全货机 16），非活跃跳过 4907，无效跳过 253
当前为预览模式；确认统计后使用 -apply 写入数据库
```

### 3. 高级用法（指定文件、版本或组织）

如需同步外部特定路径的更新文件或切换组织：

```bash
# 同步指定路径的船公司 Excel 并写入指定组织
go -C server run ./cmd/sync-shipping-lines -source /path/to/custom_liner.xlsx -org-code HQ -apply

# 同步指定路径的 UN/LOCODE ZIP 包并显式指定版本
go -C server run ./cmd/sync-unlocode -source /path/to/loc252csv.zip -release 2025-2 -apply

# 同步指定路径的 OpenFlights 航司数据
go -C server run ./cmd/sync-airlines -source /path/to/airlines.dat -apply
```

---

## 🛠 数据结构与解析规则说明

### 1. UN/LOCODE 港口数据 (`loc251csv.zip`)
- **筛选口径**：UN/LOCODE 原文件包含 11 万余行全球地点（涵盖内陆点、机场、邮局、铁路站等）。系统脚本仅严格提取具有**海港运输功能（`Function[0] == '1'`）**的有效记录（共 17,524 条），确保业务数据纯净。
- **编码清洗**：兼容 UTF-8 与 Windows-1252 编码，清洗特殊重音符号。
- **关联字段**：自动生成五字码、英文名、国家二字码、坐标经纬度、运输方式标签（`SEA`）及检索关键字（用于前端联想匹配）。

### 2. SMDG 船公司主数据 (`SMDG_Liner-codes-list-20260903.xlsx`)
- **多标识对齐**：整合 SMDG Code（3位主代码）、SCAC（美国海关4字码）、BIC Code（箱主代码）。
- **国内常用名补全**：内置国内 Top 50 核心船公司（如中远海控 COSCO、马士基 MAERSK、达飞 CMA CGM、长荣 EVERGREEN 等）的官方中文全称、简称与拼音首字母检索键。
- **航运联盟标签**：自动打标 Ocean Alliance（海洋联盟）、Gemini Cooperation（双子星）、Premier Alliance（卓越联盟）、MSC 等联盟属性。
- **箱号前缀关联**：级联建立箱号前缀表（如 `MSKU`、`COSU`、`EGLV`），支持订单录入集装箱号时智能推导承运船司。

### 3. 航空公司主数据 (`airlines.dat`)
- **全球基础数据**：复用 OpenFlights 全球航司开放数据集，过滤无 IATA 二字码的失效记录，保障空运国际覆盖率。
- **主流航司与前缀字典增强**：内置 136+ 家主流客货运航司（国航 CA、南航 CZ、东航 MU、汉莎 LH、FedEx FX、UPS 5X、Cargolux CV 等）的官方中文名称、AWB 三位运单前缀（如 999、784、112、023、406、172 等）与全货机属性。
- **约束放宽**：`awb_prefix` 与 `name_zh` 均为可选字段，既兼容全球各类小型区域航司，又为骨干货代业务提供完整的单号识别支持。

### 4. 机场主数据 (OurAirports)
- **数据源**：支持从 OurAirports 官方拉取全球机场数据（或读取本地 `airports.csv`）。
- **三字码索引**：严格校验 IATA 3 字码与 ICAO 4 字码唯一性，支持按中英文及拼音联想检索。

---

## 🔄 未来官方更新与维护流程

1. **官方更新周期**：
   - UNECE UN/LOCODE 通常于**每年 6 月与 12 月**发布两次更新（如 `2025-1`、`2025-2`）。
   - SMDG 船公司与码头表通常每季度至半年发布一次修订版。
   - OpenFlights 与 OurAirports 社区持续维护。
2. **更新替换步骤**：
   - 从上方官方链接下载最新文件。
   - 替换本目录下的对应文件（如将 `loc251csv.zip` 升级为 `loc252csv.zip`）。
   - 先执行预览命令确认无结构变更与异常冲突：
     `go -C server run ./cmd/sync-unlocode -source server/seeds/loc252csv.zip -release 2025-2`
   - 执行带 `-apply` 落库，系统将自动对比 SHA-256 并平滑增量更新。
   - 随同业务代码或单测提交 Git。

