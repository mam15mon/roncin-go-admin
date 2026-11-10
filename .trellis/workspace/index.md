# 开发工作区记录

本目录保存开发者的会话索引和日志。项目任务在独立的 `.trellis/tasks/`，历史日志保留当时事实；现行业务规则请查[根协作规范](../../AGENTS.md)和[领域地图](../spec/domain/architecture-map.md)。文档及开发者说明使用中文。

## 目录与入口

```text
.trellis/
├── tasks/                     # 活跃任务及 archive/<年月>/ 历史任务
└── workspace/
    ├── index.md               # 本入口
    └── <开发者>/
        ├── index.md           # 个人状态、日志列表与会话历史
        └── journal-N.md       # 顺序追加的会话日志
```

当前已有开发者：[mam15mon](./mam15mon/index.md)。最近活动、会话数量和当前日志由个人索引的自动区段维护，不在本入口复制一份动态统计。

## 查阅与初始化

以下命令在仓库根目录运行：

```bash
python3 .trellis/scripts/get_developer.py
python3 .trellis/scripts/get_context.py --mode record
```

新开发者尚未初始化时，使用 `python3 .trellis/scripts/init_developer.py <开发者名>` 建立身份、个人目录、索引与首份日志；已有身份不通过删除或覆盖配置来切换。具体任务流程以[工作流](../workflow.md)为准。

## 记录会话

代码或文档的可验证变更先按任务提交，完成任务后由既有脚本追加会话及更新个人索引：

```bash
python3 .trellis/scripts/add_session.py \
  --title '本次会话标题' \
  --commit '<实际工作提交哈希>' \
  --summary '具体变更、验证结果与未完成事项'
```

无工作提交时可省略 `--commit` 或传 `-`，不能编造哈希；脚本会先解析提交证据再写记录。日志自动提交与任务归档由 `.trellis/config.yaml` 的相应设置控制，不把其他任务的未提交内容并入本次会话。

日志轮转阈值来自 `max_journal_lines`（未配置时默认 2000 行），由脚本维护当前日志和索引。保留自动区段标记，避免手写另一套状态；记录实际检查结果，未执行或跳过的业务验收不写通过。
