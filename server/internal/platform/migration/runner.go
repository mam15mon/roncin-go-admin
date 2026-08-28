package migration

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const advisoryLockKey int64 = 7_266_246_125_832_581_107

var migrationNamePattern = regexp.MustCompile(`^\d{14}_[a-z0-9_]+\.sql$`)

// compatibleChecksums 仅登记经过审阅的历史迁移修复。修复后的文件用于新环境，
// 已执行旧版本的环境仍可继续前进；未登记的任何校验和差异仍然立即失败。
var compatibleChecksums = map[string]map[string]struct{}{
	"20260824043000_global_exchange_rates": {
		"d50b2a09d9b4d640285f3abb43d2d9ed05e7c701a1296363b7ab3c333cc6617c": {},
	},
	"20260826150000_order_fee_finance_foundation": {
		"eec00e191b2ff7429c7469316f2b2cbc3cd77f7c98ecb2fb6373b53c4c96989a": {},
	},
	"20260829003000_dingtalk_user_authorized_notification": {
		"ae50fc1578484e1ba96f67fcaee9b088fc2e0d1e579f4fe2088c35ff8aedbd1c": {},
	},
}

type file struct {
	name     string
	version  string
	content  string
	checksum string
}

// Apply 校验并顺序执行尚未应用的 PostgreSQL 迁移。
func Apply(ctx context.Context, db *sql.DB, dir string) error {
	files, err := readFiles(dir)
	if err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("获取迁移连接: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("获取迁移锁: %w", err)
	}
	defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockKey)

	if err := ensureRevisionTable(ctx, conn); err != nil {
		return err
	}
	applied, err := readRevisions(ctx, conn)
	if err != nil {
		return err
	}
	local := make(map[string]file, len(files))
	for _, migration := range files {
		local[migration.version] = migration
	}
	latestApplied := ""
	for version, checksum := range applied {
		migration, ok := local[version]
		if !ok {
			return fmt.Errorf("数据库中存在本地缺失的迁移版本 %s", version)
		}
		if checksum != migration.checksum && !isCompatibleChecksum(version, checksum) {
			return fmt.Errorf("迁移 %s 已执行但校验和不一致", migration.name)
		}
		if version > latestApplied {
			latestApplied = version
		}
	}
	for _, migration := range files {
		if _, ok := applied[migration.version]; ok {
			continue
		}
		if migration.version < latestApplied {
			return fmt.Errorf("迁移 %s 早于数据库最新版本 %s，禁止非线性补录", migration.name, latestApplied)
		}
		if err := applyFile(ctx, conn, migration); err != nil {
			return err
		}
	}
	return nil
}

func isCompatibleChecksum(version, checksum string) bool {
	checksums, ok := compatibleChecksums[version]
	if !ok {
		return false
	}
	_, ok = checksums[checksum]
	return ok
}

func readFiles(dir string) ([]file, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取迁移目录: %w", err)
	}
	files := make([]file, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.EqualFold(entry.Name(), "README.md") {
			continue
		}
		if !migrationNamePattern.MatchString(entry.Name()) {
			return nil, fmt.Errorf("迁移文件名不合法: %s", entry.Name())
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("读取迁移 %s: %w", entry.Name(), err)
		}
		hash := sha256.Sum256(content)
		files = append(files, file{
			name:     entry.Name(),
			version:  strings.TrimSuffix(entry.Name(), ".sql"),
			content:  string(content),
			checksum: hex.EncodeToString(hash[:]),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return files, nil
}

func ensureRevisionTable(ctx context.Context, conn *sql.Conn) error {
	var exists bool
	if err := conn.QueryRowContext(ctx, "SELECT to_regclass(current_schema() || '.schema_migrations') IS NOT NULL").Scan(&exists); err != nil {
		return fmt.Errorf("检查迁移记录表: %w", err)
	}
	if exists {
		return nil
	}
	const statement = `CREATE TABLE "schema_migrations" (
  "version" character varying(255) NOT NULL PRIMARY KEY,
  "checksum" character(64) NOT NULL,
  "applied_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	if _, err := conn.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("创建迁移记录表: %w", err)
	}
	return nil
}

func readRevisions(ctx context.Context, conn *sql.Conn) (map[string]string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT "version", "checksum" FROM "schema_migrations" ORDER BY "version"`)
	if err != nil {
		return nil, fmt.Errorf("读取迁移记录: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, fmt.Errorf("解析迁移记录: %w", err)
		}
		result[version] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历迁移记录: %w", err)
	}
	return result, nil
}

func applyFile(ctx context.Context, conn *sql.Conn, migration file) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始迁移 %s: %w", migration.name, err)
	}
	if _, err := tx.ExecContext(ctx, migration.content); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("执行迁移 %s: %w", migration.name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO "schema_migrations" ("version", "checksum") VALUES ($1, $2)`,
		migration.version, migration.checksum,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("记录迁移 %s: %w", migration.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交迁移 %s: %w", migration.name, err)
	}
	return nil
}
