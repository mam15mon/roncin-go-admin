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

// Options 控制迁移执行行为。
type Options struct {
	// AllowChecksumRepair 允许把已执行迁移的记录校验和重录为当前文件校验和。
	// 仅用于开发环境自愈：迁移文件在应用到本地库之后又被修改时，重录记录即可
	// 继续前进。生产迁移（pnpm run migrate:server）不开启该选项，差异仍立即
	// 失败。重录只更新迁移记录，不会补执行文件修改对应的 DDL。
	AllowChecksumRepair bool
	// PostStep 在迁移锁释放前执行，用于保证依赖最新 Schema 的发版必要步骤
	// 不会与其他迁移进程并发。
	PostStep func(*sql.Conn) error
	// ChecksumRepaired 在每次重录校验和后回调，供调用方输出提示。
	ChecksumRepaired func(version, oldChecksum, newChecksum string)
}

type file struct {
	name     string
	version  string
	content  string
	checksum string
}

// Apply 校验并顺序执行尚未应用的 PostgreSQL 迁移。
func Apply(ctx context.Context, db *sql.DB, dir string) error {
	return ApplyWithOptions(ctx, db, dir, Options{})
}

// ApplyWithOptions 按选项校验并顺序执行尚未应用的 PostgreSQL 迁移。
func ApplyWithOptions(ctx context.Context, db *sql.DB, dir string, opts Options) error {
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
	for _, version := range sortedVersions(applied) {
		checksum := applied[version]
		migration, ok := local[version]
		if !ok {
			return fmt.Errorf("数据库中存在本地缺失的迁移版本 %s", version)
		}
		if checksum != migration.checksum {
			if !opts.AllowChecksumRepair {
				return fmt.Errorf("迁移 %s 已执行但校验和不一致，开发环境可运行 pnpm run migrate:dev 重录校验和后继续", migration.name)
			}
			if err := repairChecksum(ctx, conn, migration, checksum, opts); err != nil {
				return err
			}
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
	if opts.PostStep != nil {
		if err := opts.PostStep(conn); err != nil {
			return fmt.Errorf("执行迁移后步骤: %w", err)
		}
	}
	return nil
}

func sortedVersions(applied map[string]string) []string {
	versions := make([]string, 0, len(applied))
	for version := range applied {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return versions
}

func repairChecksum(ctx context.Context, conn *sql.Conn, migration file, oldChecksum string, opts Options) error {
	result, err := conn.ExecContext(ctx,
		`UPDATE "schema_migrations" SET "checksum" = $1 WHERE "version" = $2 AND "checksum" = $3`,
		migration.checksum, migration.version, oldChecksum,
	)
	if err != nil {
		return fmt.Errorf("重录迁移 %s 校验和: %w", migration.name, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("确认重录迁移 %s 校验和: %w", migration.name, err)
	}
	if affected == 0 {
		return fmt.Errorf("重录迁移 %s 校验和失败：迁移记录已被并发修改", migration.name)
	}
	if opts.ChecksumRepaired != nil {
		opts.ChecksumRepaired(migration.version, oldChecksum, migration.checksum)
	}
	return nil
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
