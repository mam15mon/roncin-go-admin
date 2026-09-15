package data

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

// 角色库只归属工作台（总部/公司）节点：部门与团队沿组织树共享所属工作台的
// 角色库。本文件提供迁移期的存量归一与自检，父链解析复用登录/切换工作台同一
// 口径（workspaceAncestorID），不另写一套组织树规则。

// roleNonWorkspaceOrgKinds 是角色库不允许锚定的组织类型，取值与工作台判定
// （isWorkspaceKind）互补，避免在 SQL 中另写一份类型字面量。
var roleNonWorkspaceOrgKinds = []string{string(organization.KindDepartment), string(organization.KindTeam)}

// roleAnchorViolation 是一条锚定在部门/团队节点的角色记录。
type roleAnchorViolation struct {
	id             uuid.UUID
	code           string
	organizationID uuid.UUID
}

// roleAnchorUpdate 是角色锚点归一后的目标组织。
type roleAnchorUpdate struct {
	id             uuid.UUID
	organizationID uuid.UUID
}

// BackfillRoleWorkspaceAnchors 幂等归一存量角色锚点：把锚定在部门/团队的角色改挂到
// 其最近的总部/公司祖先，并在归一后断言角色表不再存在部门/团队锚定行。
// 组织树断链、成环导致无法解析工作台，或归一与既有 (organization_id, code) 唯一索引
// 冲突时，直接返回错误终止迁移，不做静默丢弃、重命名或停用兜底。
func BackfillRoleWorkspaceAnchors(ctx context.Context, database transactionStarter) error {
	return runTransaction(func() (*sql.Tx, error) {
		return database.BeginTx(ctx, nil)
	}, func(tx *sql.Tx) error {
		violations, err := loadRoleAnchorViolations(ctx, tx)
		if err != nil {
			return err
		}
		if len(violations) > 0 {
			nodes, err := loadRoleBackfillOrganizationNodes(ctx, tx)
			if err != nil {
				return err
			}
			updates := make([]roleAnchorUpdate, 0, len(violations))
			for _, item := range violations {
				target := workspaceAncestorID(nodes, item.organizationID)
				if target == uuid.Nil {
					return fmt.Errorf("角色 %s(%s) 锚定在无法解析出工作台的组织 %s，请先修复组织树后重试", item.code, item.id, item.organizationID)
				}
				updates = append(updates, roleAnchorUpdate{id: item.id, organizationID: target})
			}
			if err := applyRoleAnchorUpdates(ctx, tx, updates); err != nil {
				return err
			}
		}
		return assertRoleAnchorsOnWorkspace(ctx, tx)
	})
}

// loadRoleBackfillOrganizationNodes 读取组织的最小投影（ID/父节点/类型）用于父链解析，
// 与 authOrganizationNodes 的字段口径一致。
func loadRoleBackfillOrganizationNodes(ctx context.Context, tx *sql.Tx) (map[uuid.UUID]authOrgNode, error) {
	rows, err := tx.QueryContext(ctx, `SELECT "id"::text, "parent_id"::text, "kind" FROM "organizations"`)
	if err != nil {
		return nil, fmt.Errorf("读取组织树失败: %w", err)
	}
	defer rows.Close()
	nodes := make(map[uuid.UUID]authOrgNode)
	for rows.Next() {
		var rawID, kind string
		var rawParentID sql.NullString
		if err := rows.Scan(&rawID, &rawParentID, &kind); err != nil {
			return nil, fmt.Errorf("解析组织树失败: %w", err)
		}
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("解析组织 ID %q: %w", rawID, err)
		}
		node := authOrgNode{ID: id, Kind: kind}
		if rawParentID.Valid && rawParentID.String != "" {
			parentID, err := uuid.Parse(rawParentID.String)
			if err != nil {
				return nil, fmt.Errorf("解析组织 %s 的上级组织 ID %q: %w", id, rawParentID.String, err)
			}
			node.ParentID = &parentID
		}
		nodes[id] = node
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历组织树失败: %w", err)
	}
	return nodes, nil
}

// loadRoleAnchorViolations 读取锚定在部门/团队节点的角色，按 ID 排序保证归一顺序稳定。
func loadRoleAnchorViolations(ctx context.Context, tx *sql.Tx) ([]roleAnchorViolation, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT r."id"::text, r."code", r."organization_id"::text
FROM "roles" AS r
JOIN "organizations" AS o ON o."id" = r."organization_id"
WHERE o."kind" = ANY($1::text[])
ORDER BY r."id"`, roleNonWorkspaceOrgKinds)
	if err != nil {
		return nil, fmt.Errorf("读取部门/团队锚定角色失败: %w", err)
	}
	defer rows.Close()
	violations := make([]roleAnchorViolation, 0)
	for rows.Next() {
		var rawID, code, rawOrganizationID string
		if err := rows.Scan(&rawID, &code, &rawOrganizationID); err != nil {
			return nil, fmt.Errorf("解析部门/团队锚定角色失败: %w", err)
		}
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("解析角色 ID %q: %w", rawID, err)
		}
		organizationID, err := uuid.Parse(rawOrganizationID)
		if err != nil {
			return nil, fmt.Errorf("解析角色 %s 的锚定组织 ID %q: %w", id, rawOrganizationID, err)
		}
		violations = append(violations, roleAnchorViolation{id: id, code: code, organizationID: organizationID})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历部门/团队锚定角色失败: %w", err)
	}
	return violations, nil
}

// applyRoleAnchorUpdates 批量改写角色锚点；唯一索引冲突时原样返回错误终止迁移，
// 不自动改名或停用，避免掩盖需要人工确认的配置冲突。
func applyRoleAnchorUpdates(ctx context.Context, tx *sql.Tx, updates []roleAnchorUpdate) error {
	placeholders := make([]string, 0, len(updates))
	arguments := make([]any, 0, len(updates)*2)
	for index, item := range updates {
		parameter := index*2 + 1
		placeholders = append(placeholders, fmt.Sprintf("($%d::uuid, $%d::uuid)", parameter, parameter+1))
		arguments = append(arguments, item.id.String(), item.organizationID.String())
	}
	statement := fmt.Sprintf(`
UPDATE "roles" AS target
SET "organization_id" = source.organization_id
FROM (VALUES %s) AS source(id, organization_id)
WHERE target."id" = source.id`, strings.Join(placeholders, ", "))
	if _, err := tx.ExecContext(ctx, statement, arguments...); err != nil {
		return fmt.Errorf("归一角色工作台锚点失败（目标工作台可能存在同编码角色冲突，请先人工确认后重试）: %w", err)
	}
	return nil
}

// assertRoleAnchorsOnWorkspace 是迁移后的自检断言：角色表不允许存在部门/团队锚定行，
// 违反时返回错误终止迁移。
func assertRoleAnchorsOnWorkspace(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
SELECT r."code", o."kind"
FROM "roles" AS r
JOIN "organizations" AS o ON o."id" = r."organization_id"
WHERE o."kind" = ANY($1::text[])
ORDER BY r."id"
LIMIT 5`, roleNonWorkspaceOrgKinds)
	if err != nil {
		return fmt.Errorf("自检角色工作台锚点失败: %w", err)
	}
	defer rows.Close()
	remaining := make([]string, 0)
	for rows.Next() {
		var code, kind string
		if err := rows.Scan(&code, &kind); err != nil {
			return fmt.Errorf("解析自检结果失败: %w", err)
		}
		remaining = append(remaining, fmt.Sprintf("%s(%s)", code, kind))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历自检结果失败: %w", err)
	}
	if len(remaining) > 0 {
		return fmt.Errorf("迁移后自检失败：仍存在锚定在部门/团队的角色 %s", strings.Join(remaining, ", "))
	}
	return nil
}
