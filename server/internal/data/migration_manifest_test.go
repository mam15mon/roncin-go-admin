package data

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	entmigrate "github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
)

var (
	createTablePattern = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+"([^"]+)"`)
	dropTablePattern   = regexp.MustCompile(`(?i)DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?"([^"]+)"`)
	referencePattern   = regexp.MustCompile(`(?i)REFERENCES\s+"([^"]+)"`)
)

func TestMigrationManifestCoversEntSchema(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("读取迁移目录失败: %v", err)
	}
	sort.Strings(files)

	created := make(map[string]int)
	seen := make(map[string]bool)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读取迁移文件 %s 失败: %v", file, err)
		}
		for _, statement := range strings.Split(string(content), ";") {
			if match := dropTablePattern.FindStringSubmatch(statement); match != nil {
				delete(created, match[1])
				delete(seen, match[1])
				continue
			}
			if match := createTablePattern.FindStringSubmatch(statement); match != nil {
				created[match[1]]++
				seen[match[1]] = true
			}
			for _, match := range referencePattern.FindAllStringSubmatch(statement, -1) {
				if !seen[match[1]] {
					t.Fatalf("迁移 %s 在目标表创建前引用了 %s", filepath.Base(file), match[1])
				}
			}
		}
	}

	for _, table := range entmigrate.Tables {
		if created[table.Name] != 1 {
			t.Errorf("Ent 表 %s 在迁移链中应创建一次，实际为 %d 次", table.Name, created[table.Name])
		}
		delete(created, table.Name)
	}
	for table, count := range created {
		t.Errorf("迁移链包含 Ent Schema 之外的表 %s，创建 %d 次", table, count)
	}
}
