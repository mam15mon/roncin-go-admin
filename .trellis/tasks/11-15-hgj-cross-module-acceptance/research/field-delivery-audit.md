# F01–F45字段交付核对

本表将批准采纳表映射到实际实现与报告。最后一组已完成实现、两侧独立复核及真实API/浏览器/CSV/财务验收；既有能力
保留相邻验证，不把保留字段包装为新增功能。最终命令见cross-module-root-validation。

报告简称：方向组`archive/2026-11/11-15-hgj-order-direction/research`，
航空正文组`...hgj-air-documents/research`，SI组`...hgj-sea-import/research`，
AI组`...hgj-air-import/research`，PDF组`...hgj-air-waybill-pdf/research`。
全部路径在`.trellis/tasks/`下，各报告保留真实命令与失败修复记录。

| 字段 | 实际归属与证据 | 最终补验状态 |
| --- | --- | --- |
| F01 | Order四类类型/方向合法配对，方向/航空/SI/AI真实PG与HTTP；四类注册adapter | 最终四类真实财务建票通过 |
| F02 | 主单不可变business_type、方向唯一索引/迁移回填拒绝、候选/按ID/完整成员；方向组core/review | 前置通过；四类台账真实主单搜索通过 |
| F03 | 沿用真实NumberRule/事务序列，四类实际创建及幂等；方向与SI/AI浏览器 | 四类真实号码、过期版本拒绝无写入通过；原子性沿前置PG |
| F04 | 同公司Partner；销售/客服沿历史确认恒只读+新建必填/客户档案权威配置，操作等可编辑岗保留手改，创建人权威；真实权限PG及最终四类客户前置拒绝 | 两公司/单类角色HTTP及只读浏览器整轮通过 |
| F05 | 既有shipmentMode，公共UI/创建更新/清空；方向组air-common及航空映射 | 前置通过 |
| F06 | 客户/内部业务号100字符，biz及控件；Root发现Ent字节口径缺陷，已交本组最小修 | 中文边界真实PG成功及+1拒绝通过 |
| F07 | tradeTerm公共选择/映射/显式清空，不自动带费；air-common | 前置通过 |
| F08 | 业务PREPAID/COLLECT与独立AWB付款印字；真实UI与PDF数值 | 前置通过 |
| F09 | bookingAgentId本票供应商候选/更新清空，不代换单代理；air-common、SI/AI | 前置通过 |
| F10 | foreignAgentId与单证文字各自保存；公共adapter及主分内容PG | 前置通过 |
| F11 | 海运shippingAgentId真实公共字典，SI适用；空运不增船代 | 前置海运相邻通过 |
| F12 | ImportHandling.exchangeAgentId独立供应商、公司/角色校验；SI core/前端/浏览器 | 前置通过 |
| F13 | EXCHANGE完成实际时间，纳秒wire保留、真实编辑ISO；SI/AI五服务事件/前端 | 前置通过 |
| F14 | contractNo100、公共UI/清空映射；发现Ent字符口径同F06 | 100字符中文合约真实PG与+1拒绝通过 |
| F15 | 货值币种成对/零/4位精度，UI金额控件/adapter、biz及真实wire | 最终四类真实样票及CSV核对通过 |
| F16 | 保费币种与AWB保险声明独立；air-common、真实主分PDF | 前置通过 |
| F17 | receivedAt/cargoReadyAt独立Dayjs/ISO、清空/无回退；公共adapter测试 | 前置通过 |
| F18 | factoryName200文本，公共UI/清空；发现Ent字符口径同F06 | 200中文工厂真实PG与+1拒绝通过 |
| F19 | 四类notes各1000独立映射/多行；Ent字节口径本组修 | 1000中文备注真实PG与+1拒绝通过 |
| F20 | 八角色、服务端创建人、同公司候选与提成归属；方向/SI/AI权限测试 | 四类真实销售提成各6、确认及PAID通过 |
| F21 | 固定五环节undertaken/完成/并行/撤回原因/事件/完结门禁，Order版本；SI/AI PG+浏览器 | 最终SI/AI真实2/5承接完成、完结审核与财务通过 |
| F22 | 海运真实箱计划/独占共享箱归属及守恒，SI扩展；SE相邻PG | 前置通过 |
| F23 | 委托与实际CargoItem独立、主单完整汇总；航空PG及AI浏览器150/180/210 | 前置真实PG重量费用来源通过，最终经营费用显式录入与货量独立 |
| F24 | UN/CLASS/截申报独立，Root发现航空取消后optional省略保旧值 | AE/AI真实HTTP设→省略保持→显式空/PG0清除及精确版本通过 |
| F25 | PG_I/II/III与可空语义，普通航空映射；取消PG显式零缺口同F24 | 真实HTTP与PG清空回归通过，保持既有PG枚举 |
| F26 | ORDER/MASTER、PLANNED/ACTUAL、指纹与精确有理数；航空PG、AI真实UI | 前置真实PG费用采用与过期保护通过；最终财务显式录费不反写重量 |
| F27 | 不采纳HG快捷eval/错误KG-CBM切换，保留明确除数/分子分母 | 代码范围与研究边界核对通过 |
| F28 | 真实航司/机场/主单权威单航班时刻、共享修改预览/版本；AE/AI PG与UI | 最终四类ETD出口/ETA进口中国业务日、报表读链通过 |
| F29 | 真正多航段后续；纸面路线位仅印字 | 无新运输引擎，边界保持 |
| F30 | SeaMaster/TransportExecution共同段与本票最终目的港，SI复用；core/review PG | 最终四类台账号实际搜索及海运中文船名快照通过 |
| F31 | 各截止时刻独立、不拿换单时间复用；海空现合同/adapter与wire | 前置通过 |
| F32 | AE/AI DIRECT真实无HAWB/单成员，HOUSE独立分单/历史，普通更新禁互转/换主；航空/AI PG+PDF浏览器 | 最终四类各DIRECT财务链通过；双模式全矩阵沿前置 |
| F33 | 独立MAWB/HAWB正文，party5000及历史/hash/版本；航空PG、PDF15页party全文 | 前置通过 |
| F34 | 航空marks/description各10000、handling/clauses/remarks5000，不串本票货描；PDF14页全文 | 前置通过 |
| F35 | 航空长文上限与超限，海运旧长度不缩；真实PDF64页专项 | 前置通过 |
| F36 | 保存正文件重尺/运价与经营费独立，0/空/非零；各200行真实19页PDF | 前置通过 |
| F37 | 声明NVD/NCV/XXX/AMOUNT、PPD/COLL、纯日期及真主体；六份真实16页PDF | 前置通过 |
| F38 | 同Blob/lazy/同域字体/指纹与版本输入保护；PDF root/editor/browser报告及build | 前置通过，HBL四版+SI归档相邻 |
| F39 | 报价自动带费后续，无假引用或未经确认价格带入 | 边界保持 |
| F40 | 不采纳自由服务类型/业务标签/流程平台，只固定五服务 | 前五组源/迁移和页面无恢复 |
| F41 | 专用完整筛选CSV，无分页同读快照/授权列/排序/onLoad采纳 | 同快照/撤权保护及四类真实各205行、生产205行、只读206行授权CSV通过 |
| F42 | 批量导入后续，不提供假导入按钮 | 边界保持 |
| F43 | 复杂空运拆拼改配后续，既有SE高级能力保留；方向组和海运相邻PG | 前置真实PG及最终完整门禁出口相邻通过 |
| F44 | 仓库安检/库存/自动配载/航司海关税务通道后续，登记不称外部办理 | 边界保持 |
| F45 | 另五类只调研，见final-research-audit及父other-business-matrix | AC5/V19研究边界复核完成 |

最终状态以各专项真实报告及私有facts/CSV/PDF为依据。新发现跨层缺口已由
所属实现/独立检查者修复，触发与相邻验证通过；研究中不采纳和后续项仍明确保持范围。
