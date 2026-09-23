# 丰富费用科目系统模板种子目录

## Goal

系统费用模板目录（`fee_setting_templates`）当前仅 10 条，无法覆盖实际货代业务。依据用户提供的
参考科目清单（约 141 行，来自既有系统导出），经清洗、去重、补英文后形成权威种子清单
（含现有 10 条共 **122 条**），通过一个正式 SQL 迁移落库，并配套补充费用类别、异常情况、
计费单位主数据，最后把新增科目追加到现有公司（`fee_settings`），使分公司本地目录即刻可用。

## 背景与机制（已与用户核实）

- 「总部导入」= 系统初始目录 `fee_setting_templates`（全局一份，fee_code 唯一）；
  「分公司本地追加」= `fee_settings` 按公司隔离，同码时本地优先。
- 模板目前**不会自动下发**到已存在的公司；全新生产库的模板表与计费单位表均为空
  （既有 10 条模板是 20260922 迁移从旧共享目录转换而来，计费单位仅由 sync-dev 种植）。
  因此本迁移必须同时承担：全新库完整种子 + 既有库增量补充。
- 用户已确认的四项默认决策：
  1. 税费/财务调整类科目（29 条）收录，单独挂「税费」「财务调整」类别；
  2. 同名不同税率档位（0%/6%/9% 等）保留拆分（发票口径需要）；
  3. 代码风格保留参考清单的拼音码习惯，仅修复冲突与乱码；现有 10 条英文码保留不动；
  4. 参考清单等价科目以别名并入现有 10 条，不重复建码。

## 需求

1. 新增正式迁移 `server/migrations/<ts>_fee_catalog_seed.sql`，单个事务内完成：
   - 补种计费单位（CONT/BL/CBM/TON/SET/DOC + 新增 DAY 天），`ON CONFLICT (code) DO NOTHING`；
   - 补种费用类别 +5：PORT_OPS 港口操作、AIR 空运、TAX 税费、FIN_ADJ 财务调整、MISC 其他；
   - 补种异常情况：新增 ABN-WAITING 待时、ABN-OVERNIGHT 压夜、ABN-INSPECTION 查验，
     并幂等补种 ABN-CUSTOMER-CANCEL / ABN-ROLL-REASSIGN（全新库缺行，模板 KC 引用其 ID）；
   - 将 122 条模板 INSERT 进 `fee_setting_templates`，`ON CONFLICT (fee_code) DO NOTHING`
     （开发库已有 10 条保持原样；全新库一次种全）；
   - 将现有模板 THC/STORAGE 的费用类别改挂 PORT_OPS（仅模板行，不动公司副本）；
   - 按 20260922 的复制模式把模板追加到每个 `kind='company'` 公司：同码跳过、
     应税劳务按 (公司, 名称) 找不到则建、复制模板全部字段含 search_keywords。
2. search_keywords 必须与 Ent 钩子 `searchtext.Build(name_zh, name_en, alias_name)` 输出
   完全一致（原文大写 + 全拼连写 + 分写 + 首字母）；SQL 无法调用钩子，实现时用一次性
   Go 程序预计算后写入 SQL，程序本身不提交。
3. 新增应税劳务名称（模板存文本）：国际航空运输服务（0%）、经纪代理服务（6%）。
4. 不修改任何 Go 契约、权限码、前端代码；模板管理界面无需变更。

## 权威种子清单（122 条）

分组列出：代码 | 中文名 | 英文名 | 别名 | 类别 | 币种 | 单位 | 税率 | 应税劳务 | 异常。
「现有」标记的 10 条已在库中，迁移对其仅做 ON CONFLICT 跳过（THC/STORAGE 另做类别调整）。

### ① 订舱 BOOKING（基数 1000）

| 代码 | 中文名 | 英文名 | 别名 | 币种 | 单位 | 税率 | 应税劳务 | 异常 | 备注 |
|---|---|---|---|---|---|---|---|---|---|
| OF | 海运费 | Ocean Freight | | USD | CONT | 0 | 国际海运运费 | | 现有 |
| BL | 提单费 | B/L Fee | | USD | BL | 0 | 国际海运运费 | | |
| DCF | 订舱费 | Booking Fee | | CNY | BL | 6 | 海运代理订舱服务 | | |
| CDF | 舱单费 | Manifest Filing Fee | | CNY | BL | 6 | 海运代理订舱服务 | | |
| AMS | AMS 舱单费（美线） | AMS Filing Fee | | USD | BL | 0 | 海运代理订舱服务 | | |
| AFR | AFR 舱单费（日本） | AFR Filing Fee | | USD | BL | 0 | 海运代理订舱服务 | | |
| ENS | ENS 舱单费（欧盟） | ENS Filing Fee | | USD | BL | 0 | 海运代理订舱服务 | | |
| VGM | 重量验证费 | VGM Fee | | CNY | CONT | 6 | 港口操作及港杂服务 | | 现有 |
| DET | 滞箱费 | Container Detention Fee | | CNY | DAY | 0 | 港口操作及港杂服务 | | 原码 ZXF 冲突改码 |
| KC | 空舱费 | Dead Freight | 亏仓费 | CNY | BL | 0 | 海运代理订舱服务 | ABN-CUSTOMER-CANCEL | 空舱/亏仓合并 |
| FDFW | 放单服务费 | Release Order Fee | | CNY | BL | 0 | 海运代理订舱服务 | | 原码为中文 |
| DLF | 代理费 | Agency Fee | | CNY | BL | 0 | 海运代理订舱服务 | | |
| SXF | 手续费 | Handling Charge | | CNY | BL | 0 | 海运代理订舱服务 | | |

### ② 制单 DOC_PREP（基数 1100）

| 代码 | 中文名 | 英文名 | 别名 | 币种 | 单位 | 税率 | 应税劳务 | 备注 |
|---|---|---|---|---|---|---|---|---|
| DOC | 文件费 | Documentation Fee | | CNY | BL | 6 | 海运代理订舱服务 | 现有 |
| TLX | 电放费 | Telex Release Fee | | CNY | BL | 6 | 海运代理订舱服务 | 现有 |
| SEAL | 封条费 | Seal Fee | 封志费 | CNY | CONT | 6 | 海运代理订舱服务 | 现有，并入 FZF |
| EIR | 打单费 | Equipment Interchange Receipt | 设备交接单费 | CNY | CONT | 6 | 海运代理订舱服务 | 现有，并入 DDF/SBJJDF；单位随 sync-dev 为箱 |
| GDF | 改单费 | Amendment Fee | | EUR | BL | 0 | 海运代理订舱服务 | |
| LDF | 联单费 | Combined Documentation Fee | | CNY | BL | 6 | 海运代理订舱服务 | |
| TDF | 调单费 | Document Retrieval Fee | | CNY | BL | 0 | 海运代理订舱服务 | 原码为中文 |
| KDF | 快递费 | Courier Fee | | CNY | BL | 0 | 海运代理订舱服务 | |

### ③ 换单 DOCUMENT_EXCHANGE（基数 1150）

| 代码 | 中文名 | 英文名 | 别名 | 币种 | 单位 | 税率 | 应税劳务 | 备注 |
|---|---|---|---|---|---|---|---|---|
| HDF | 换单费 | Delivery Order Fee | 换单服务 | CNY | BL | 6 | 海运代理订舱服务 | 并入「换单服务 HDFW」 |
| HKG | 香港申报费 | Hong Kong Declaration Fee | | CNY | BL | 6 | 报关报检代理服务 | |

### ④ 报关 CUSTOMS_EXPORT / 清关 CUSTOMS_IMPORT / 买单 DOC_BUY（基数 1200）

| 代码 | 中文名 | 英文名 | 别名 | 币种 | 单位 | 税率 | 应税劳务 | 类别 | 备注 |
|---|---|---|---|---|---|---|---|---|---|
| CUSTOMS | 代理报关费 | Customs Declaration Fee | | CNY | BL | 6 | 报关报检代理服务 | CUSTOMS_EXPORT | 现有 |
| BGF | 报关费（无票） | Customs Declaration Fee (No Invoice) | 报关费 | CNY | BL | 0 | 报关报检代理服务 | CUSTOMS_EXPORT | 与 6% 档拆分 |
| HZQD | 核注清单费 | Bonded List Filing Fee | | CNY | BL | 6 | 报关报检代理服务 | CUSTOMS_EXPORT | |
| CYF | 查验费 | Customs Inspection Fee | 查验服务 | CNY | BL | 0 | 报关报检代理服务 | CUSTOMS_EXPORT | 异常 ABN-INSPECTION |
| HGFK | 海关罚款 | Customs Penalty | | CNY | BL | 0 | 报关报检代理服务 | CUSTOMS_EXPORT | |
| ZBJ | 滞报金 | Late Declaration Penalty | | CNY | BL | 0 | 报关报检代理服务 | CUSTOMS_EXPORT | |
| MD | 买单 | Export Doc Purchase | | CNY | BL | 0 | 报关报检代理服务 | DOC_BUY | 原码 111 |
| T1 | T1 转关费（欧盟） | EU T1 Transit Fee | | EUR | BL | 0 | 报关报检代理服务 | CUSTOMS_IMPORT | |
| DUTY | 关税 | Customs Duty | | CNY | BL | 0 | 报关报检代理服务 | CUSTOMS_IMPORT | 原码 123 |

### ⑤ 报检 INSPECTION（基数 1250）

| 代码 | 中文名 | 英文名 | 币种 | 单位 | 税率 | 应税劳务 | 备注 |
|---|---|---|---|---|---|---|---|
| BJF | 报检费 | Inspection Declaration Fee | CNY | BL | 6 | 报关报检代理服务 | |
| SJF | 商检费 | Commodity Inspection Fee | CNY | BL | 0 | 报关报检代理服务 | |
| JYF | 检验费 | Inspection & Testing Fee | USD | BL | 0 | 报关报检代理服务 | |
| JD | 鉴定费（6%） | Appraisal Fee (6%) | CNY | BL | 6 | 报关报检代理服务 | |
| JDF | 鉴定费 | Appraisal Fee | CNY | BL | 0 | 报关报检代理服务 | |
| XDF | 消毒费 | Disinfection Fee | CNY | BL | 0 | 报关报检代理服务 | |
| YDF | 验电费 | Electrical Inspection Fee | CNY | BL | 0 | 报关报检代理服务 | |

### ⑥ 办证 CERTIFICATE / 熏蒸 FUMIGATION / 保险 INSURANCE（基数 1280）

| 代码 | 中文名 | 英文名 | 币种 | 单位 | 税率 | 应税劳务 | 类别 | 备注 |
|---|---|---|---|---|---|---|---|---|
| COO | 代办产地证 | Certificate of Origin | CNY | BL | 0 | 报关报检代理服务 | CERTIFICATE | 原两条 COO 合并 |
| DJF | 登记费 | Registration Fee | USD | BL | 0 | 报关报检代理服务 | CERTIFICATE | |
| XZF | 熏蒸费 | Fumigation Fee | CNY | BL | 0 | 报关报检代理服务 | FUMIGATION | |
| BXF | 保险费 | Insurance Fee | CNY | BL | 0 | 海运代理订舱服务 | INSURANCE | 原码 BXF6% 拆档 |
| BXF6 | 保险费（6%） | Insurance Fee (6%) | CNY | BL | 6 | 海运代理订舱服务 | INSURANCE | |

### ⑦ 拖车 TRUCKING（基数 1300，应税劳务均为集装箱陆路运输服务）

| 代码 | 中文名 | 英文名 | 别名 | 币种 | 单位 | 税率 | 异常 | 备注 |
|---|---|---|---|---|---|---|---|---|
| TRUCK | 集装箱拖车费 | Container Trucking Fee | 拖车费 | CNY | CONT | 9 | | 现有，并入 TCF |
| YF9 | 运费（9%） | Inland Freight (9%) | | CNY | BL | 9 | | |
| YF6 | 运费（6%） | Inland Freight (6%) | | CNY | BL | 6 | | |
| PY | 运费（普票 6%） | Inland Freight (General Invoice) | | CNY | BL | 6 | | 原「6%普票-运费」 |
| KCF9 | 卡车费（9%） | Trucking Fee (9%) | | CNY | BL | 9 | | 原码 KCF9% |
| GNYDL6 | 国内运输代理费（6%） | Domestic Forwarding Fee (6%) | | CNY | BL | 6 | | 原码 GNYSDLF6% 过长 |
| SHF | 送货费 | Delivery Fee | | CNY | BL | 0 | | |
| SHF9 | 送货费（9%） | Delivery Fee (9%) | | CNY | BL | 9 | | |
| THF | 提货费 | Pick-up Fee | | CNY | BL | 0 | | |
| P | 派送费 | Delivery Fee (Domestic) | | CNY | BL | 0 | | |
| RFT | 铁路运费 | Rail Freight | | USD | BL | 0 | | |
| DSF | 待时费 | Waiting Time Fee | | CNY | BL | 0 | ABN-WAITING | |
| YYF | 压夜费 | Overnight Detention Fee | | CNY | BL | 0 | ABN-OVERNIGHT | |
| YCF | 压车费 | Vehicle Detention Fee | | CNY | BL | 0 | | |
| GG | 过港费 | Cross-Harbor Transfer Fee | | CNY | BL | 0 | | |

### ⑧ 港口操作 PORT_OPS（基数 1400，应税劳务均为港口操作及港杂服务）

| 代码 | 中文名 | 英文名 | 别名 | 币种 | 单位 | 税率 | 备注 |
|---|---|---|---|---|---|---|---|
| THC | 码头操作费 | Terminal Handling Charge | | CNY | CONT | 6 | 现有，类别改挂本组 |
| STORAGE | 码头堆存费 | Port Storage Fee | 堆存费 | CNY | CONT | 6 | 现有，类别改挂本组，并入 DCF |
| GZF | 港杂费 | Port Miscellaneous Fee | | CNY | BL | 0 | |
| GZF6 | 港杂费（6%） | Port Miscellaneous Fee (6%) | | CNY | BL | 6 | 原码为中文 |
| DMF6 | 地面服务费（6%） | Ground Handling Fee (6%) | 地面操作费 | CNY | BL | 6 | |
| CZF | 操作费 | Local Handling Fee | | CNY | BL | 0 | |
| ZX | 装卸费 | Loading/Unloading Fee | | CNY | BL | 0 | 原码 ZXF 冲突改码 |
| TSC | 特殊操作费 | Special Handling Fee | | CNY | BL | 0 | |
| JBF | 加班费 | Overtime Fee | | CNY | BL | 0 | |
| RGF | 人工费 | Labor Fee | | USD | BL | 0 | |

### ⑨ 仓储 WAREHOUSING（基数 1450，应税劳务均为港口操作及港杂服务）

| 代码 | 中文名 | 英文名 | 币种 | 单位 | 税率 | 备注 |
|---|---|---|---|---|---|---|
| CCF | 仓储费 | Warehouse Storage Fee | CNY | DAY | 0 | |
| CCF6 | 仓储费（6%） | Warehouse Storage Fee (6%) | CNY | DAY | 6 | |
| DDCCF | 代垫仓储费 | Advance Storage Fee | CNY | DAY | 0 | |
| DDCCF6 | 代垫仓储费（6%） | Advance Storage Fee (6%) | CNY | DAY | 6 | |
| JCF | 进仓费 | Inbound Warehousing Fee | CNY | BL | 0 | |
| CRKF | 出入库费 | In/Out Warehousing Fee | CNY | BL | 0 | |
| ZZF | 转栈费 | Yard Transfer Fee | CNY | BL | 0 | |
| TK | 退库费 | Warehouse Return Fee | CNY | BL | 0 | |
| XXF | 洗箱费 | Container Cleaning Fee | CNY | CONT | 0 | |
| REP | 修箱费 | Container Repair Fee | CNY | CONT | 0 | 原码 XXF1 |

### ⑩ 内装 STUFFING（基数 1500，应税劳务均为港口操作及港杂服务）

| 代码 | 中文名 | 英文名 | 币种 | 单位 | 税率 | 备注 |
|---|---|---|---|---|---|---|
| ZXF | 装箱费 | Stuffing Fee | CNY | BL | 0 | 原码与装卸/滞箱冲突 |
| TXF | 掏箱费 | Devanning Fee | CNY | BL | 0 | |
| KXF | 开箱费 | Container Opening Fee | CNY | BL | 0 | |
| FXF | 放箱费 | Container Release Fee | CNY | BL | 0 | |
| JHTP | 交换托盘 | Pallet Exchange Fee | USD | BL | 0 | |

### ⑪ 空运 AIR（基数 1550，应税劳务均为新增「国际航空运输服务 0%」）

| 代码 | 中文名 | 英文名 | 币种 | 单位 | 税率 | 类别 | 备注 |
|---|---|---|---|---|---|---|---|
| KYF | 空运费 | Air Freight | CNY | BL | 0 | AIR | |
| AWB | 运单费 | Air Waybill Fee | EUR | BL | 0 | AIR | |
| HZF | 航站费 | Airport Terminal Fee | USD | BL | 0 | AIR | |
| CSF | 安检费 | Security Screening Fee | EUR | BL | 0 | AIR | |
| AMF | 机场快递费 | Airport Messenger Fee | EUR | BL | 0 | AIR | 原英文 messanger 已修正 |
| AF | A/F 费 | A/F Fee | EUR | BL | 0 | AIR | |
| ATB | ATB 费 | ATB Fee | CNY | BL | 0 | AIR | 含义存疑保留 |
| CLC | 箱板费 | Air Pallet Fee | USD | BL | 0 | PALLET_CHARTER 包板 | |

### ⑫ 海外段 OVERSEA_SEGMENT（基数 1600，应税劳务除标注外均为国际海运运费）

| 代码 | 中文名 | 英文名 | 币种 | 单位 | 税率 | 备注 |
|---|---|---|---|---|---|---|
| DTHC | 目的港码头操作费 | Destination THC | USD | BL | 0 | |
| ETS | ETS 附加费 | EU Emission Trading Surcharge | EUR | BL | 0 | 原两条同码合并，默认 EUR |
| PU | 提货费（海外） | Pick-up Fee | EUR | BL | 0 | 即参考「提货 P/U」 |
| HDL | 操作费（海外） | Handling Fee | EUR | BL | 0 | 原码为英文单词 Handling |
| DLV | 派送费（海外） | Delivery Fee | EUR | BL | 0 | 即参考「Delivery」 |
| CMP | 合规查验费 | Compliance Check Fee | EUR | BL | 0 | |
| ATLAS | ATLAS 系统费 | ATLAS Fee | EUR | BL | 0 | |
| GENEST | GENEST 费 | GENEST Fee | CNY | BL | 0 | 含义存疑保留 |
| MYD | 贸易代理费 | Trading Agent Fee | USD | BL | 0 | |
| GJDLF | 国际货运代理费 | International Freight Forwarding Fee | CNY | BL | 0 | 原码 GJHWYSDLF 过长；劳务=海运代理订舱服务 |

### ⑬ 税费 TAX（基数 1700，应税劳务除 GST/印花税/企业所得税外均为报关报检代理服务）

| 代码 | 中文名 | 英文名 | 别名 | 币种 | 单位 | 税率 | 备注 |
|---|---|---|---|---|---|---|---|
| SJ | 税金 | Taxes & Duties | 税款 | CNY | BL | 0 | 并入「税款 SK」 |
| DDSJ | 代垫税金（12%） | Advance Tax (12%) | | CNY | BL | 12 | |
| ZZS9 | 增值税（9%） | VAT (9%) | | CNY | BL | 9 | 原三条 ZZS 同码拆分 |
| ZZS6 | 增值税（6%） | VAT (6%) | | CNY | BL | 6 | |
| ZZS3 | 增值税（9%-6%补差） | VAT Differential (9%-6%) | | CNY | BL | 3 | |
| JXS | 进项税 | Input VAT | | CNY | BL | 0 | |
| XXZZS | 销项增值税 | Output VAT | | CNY | BL | 0 | |
| YHS | 印花税 | Stamp Duty | | CNY | BL | 0 | 劳务=经纪代理服务 |
| QYSDS | 企业所得税 | Corporate Income Tax | | CNY | BL | 0 | 劳务=经纪代理服务 |
| GST | 商品服务税 | GST | | SGD | BL | 0 | 劳务=经纪代理服务 |
| ZNJ | 滞纳金 | Late Payment Surcharge | | CNY | BL | 0 | |

### ⑭ 财务调整 FIN_ADJ / 其他 MISC（基数 1800，应税劳务均为新增「经纪代理服务 6%」）

| 代码 | 中文名 | 英文名 | 别名 | 币种 | 单位 | 税率 | 类别 | 备注 |
|---|---|---|---|---|---|---|---|---|
| TZ | 账务调整 | Account Adjustment | 财务调账 | CNY | BL | 0 | FIN_ADJ | 原「财务调账/调账/账务调整」三条合并 |
| HDSY | 汇兑损益 | FX Gain/Loss | | CNY | BL | 0 | FIN_ADJ | |
| CDHP | 汇票承兑手续费（无票） | Bank Acceptance Fee (No Invoice) | | CNY | BL | 0 | FIN_ADJ | |
| XJZC | 现金支出 | Cash Disbursement | | CNY | BL | 0 | FIN_ADJ | |
| LRC | 利润分成 | Profit Share | PROFIT SHARE | CNY | BL | 0 | FIN_ADJ | 原 PS+LRF 合并 |
| YFK | 预付款 | Advance Payment | | USD | BL | 0 | FIN_ADJ | |
| BZJ | 保证金 | Deposit | | CNY | BL | 0 | FIN_ADJ | |
| DDHK | 代垫货款 | Advance Cargo Payment | | CNY | BL | 0 | FIN_ADJ | |
| QT | 其他 | Miscellaneous | | CNY | BL | 0 | MISC | |

### 主数据补充

- 计费单位：`DAY 天`（sort 70，非箱型），并幂等补种 CONT/BL/CBM/TON/SET/DOC
  （名称与 is_container_unit 与 sync-dev 保持一致：仅 CONT 为箱型）。
- 费用类别（name_en 一并提供，source=system）：
  PORT_OPS 港口操作 Port Operations（200）、AIR 空运 Air Freight（210）、
  TAX 税费 Taxes & Duties（220）、FIN_ADJ 财务调整 Finance Adjustment（230）、
  MISC 其他 Miscellaneous（240）。
- 异常情况：ABN-WAITING 待时 Waiting Time（10）、ABN-OVERNIGHT 压夜 Overnight（20）、
  ABN-INSPECTION 查验 Customs Inspection（30）；
  幂等补种 ABN-CUSTOMER-CANCEL 客户取消出运 Customer Cancellation（40）、
  ABN-ROLL-REASSIGN 甩柜改配 Roll-over Reassignment（50）。

## Acceptance Criteria

- [x] 全新库：迁移后 `fee_setting_templates` 恰好 122 条、fee_code 无重复；
      7 个应税劳务名称、7 个计费单位（含 DAY）、24 个费用类别、5 个异常情况就位。
- [x] 全新库：每个 `kind='company'` 公司的 `fee_settings` 追加至 122 条，
      应税劳务按名称在本公司创建/复用，无跨公司泄漏。
- [x] 开发库：既有 10 条模板与公司科目原样保留（THC/STORAGE 模板类别改为 PORT_OPS），
      新增 112 条模板 + 公司追加 112 条（同码跳过），订单费用引用不受影响。
- [x] 所有种子行 search_keywords 与 `searchtext.Build` 输出一致（抽查比对）。
- [x] 迁移集成测试覆盖：全新库种子计数、公司追加、同码跳过（已有本地同码科目时不覆盖）。
- [x] `pnpm run migrate:dev` 在开发库幂等重放安全（ON CONFLICT DO NOTHING）。
- [x] 不修改生成物、契约、权限码；Go 代码零改动（除测试）。

## Notes

- 参考清单原样存档于本 PRD 之上方对话记录；清洗规则：同码冲突拆码、乱码（123/111/中文码）
  改语义码、纯英文单词码转缩写、同义科目合并并保留别名检索、税率按名称档位恢复。
- GENEST/ATB/A/F 含义存疑，按用户「先保留」处理，上线后可在模板界面禁用。
