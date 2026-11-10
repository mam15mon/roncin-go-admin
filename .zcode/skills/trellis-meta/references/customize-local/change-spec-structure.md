# Change Local Spec Structure

When the user wants to change the engineering conventions AI follows, add new spec layers, or adjust monorepo package mapping, edit `.trellis/spec/` and `.trellis/config.yaml`.

## Read These Files First

1. `.trellis/config.yaml`
2. `.trellis/spec/`
3. `.trellis/workflow.md` planning artifact guidance and Phase 3.3
4. Current task `implement.jsonl` / `check.jsonl`

## Common Needs

| Need | Edit location |
| --- | --- |
| Add backend/frontend/docs/test spec layer | `.trellis/spec/<layer>/` or `.trellis/spec/<package>/<layer>/` |
| Add shared thinking guides | `.trellis/spec/guides/` |
| Adjust monorepo packages | `packages` in `.trellis/config.yaml` |
| Change default package | `default_package` in `.trellis/config.yaml` |
| Control spec scanning scope | `.trellis/config.yaml` 中的 `session.spec_scope`，不是顶层 `spec_scope` |
| Make a task read a new spec | Task `implement.jsonl` / `check.jsonl` |

## Add A Spec Layer

Single-repository example:

```text
.trellis/spec/security/
├── index.md
└── auth.md
```

Monorepo example:

```text
.trellis/spec/webapp/security/
├── index.md
└── auth.md
```

`index.md` should include:

- What code this layer applies to.
- Pre-Development Checklist.
- Quality Check.
- Links to specific guideline files.

## Update Context

Adding a spec does not mean every task automatically reads it. The current task must reference it in JSONL:

```bash
python3 ./.trellis/scripts/task.py add-context <task> implement ".trellis/spec/webapp/security/index.md" "Security conventions"
python3 ./.trellis/scripts/task.py add-context <task> check ".trellis/spec/webapp/security/index.md" "Security review rules"
```

## Change Monorepo Packages

Example `.trellis/config.yaml`:

```yaml
packages:
  webapp:
    path: apps/web
  api:
    path: apps/api
default_package: webapp
```

After editing, run:

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
```

Use this output to confirm AI can see the correct packages and spec layers.

扫描范围由 `session.spec_scope` 读取；可省略以扫描全部包，或使用包名列表、`active_task`：

```yaml
session:
  spec_scope: active_task
```

这里只展示配置位置，不替本项目修改配置；以 `.trellis/scripts/common/config.py` 的 `get_spec_scope` 为准。

## Notes

- Specs are user project conventions and can be changed according to project needs.
- Do not put temporary task information into specs; put temporary information in the task.
- Do not put long-term conventions only in agents or commands; preserve them in specs.
- After changing spec structure, check whether existing task JSONL files still point to files that exist.
