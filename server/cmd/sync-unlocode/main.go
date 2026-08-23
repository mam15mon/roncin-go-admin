package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data"
	"golang.org/x/text/encoding/charmap"
)

const unlocodeSource = "UNECE_UNLOCODE"

var (
	unlocodePattern = regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{3}$`)
	countryPattern  = regexp.MustCompile(`^[A-Z]{2}$`)
	releasePattern  = regexp.MustCompile(`^(\d{4}-[12])\s+`)
	partNames       = []string{"UNLOCODE CodeListPart1.csv", "UNLOCODE CodeListPart2.csv", "UNLOCODE CodeListPart3.csv"}
)

type options struct {
	apply            bool
	source           string
	release          string
	organizationCode string
}

type unlocodeParseSummary struct {
	RawRows        int
	ValidPorts     int
	NonPorts       int
	CountryHeaders int
	Aliases        int
	Superseded     int
	InvalidRows    int
	Withdrawn      int
	SourceHash     string
	FatalProblems  []string
}

func main() {
	ctx := context.Background()
	options := parseOptions()
	raw, err := os.ReadFile(options.source)
	if err != nil {
		fail("读取 UN/LOCODE 发布包失败", err)
	}
	files, detectedRelease, err := readCodeListFiles(raw)
	if err != nil {
		fail("解析 UN/LOCODE 发布包失败", err)
	}
	if options.release == "" {
		options.release = detectedRelease
	}
	if options.release == "" {
		fail("缺少 UN/LOCODE 版本", errors.New("发布包文件名不含版本前缀，请显式传 -release"))
	}
	rows, summary, err := parseUNLocode(files, raw)
	if err != nil {
		fail("解析 UN/LOCODE 数据失败", err)
	}
	store, cleanup, err := openStore()
	if err != nil {
		fail("初始化港口同步存储失败", err)
	}
	defer cleanup()
	conflicts, err := store.CheckPorts(ctx, options.organizationCode, unlocodeSource, rows)
	if err != nil {
		fail("检查港口同步冲突失败", err)
	}
	printSummary(options, summary, conflicts)
	if len(summary.FatalProblems) > 0 || len(conflicts) > 0 {
		fail("港口数据存在致命冲突", fmt.Errorf("源文件冲突 %d 条，数据库冲突 %d 条", len(summary.FatalProblems), len(conflicts)))
	}
	if !options.apply {
		fmt.Println("当前为预览模式；确认统计后使用 -apply 写入数据库")
		return
	}
	result, err := store.ApplyPorts(ctx, options.organizationCode, unlocodeSource, options.release, summary.SourceHash, rows)
	if err != nil {
		fail("写入港口数据失败", err)
	}
	fmt.Printf("港口同步完成：新增 %d，更新 %d，停用 %d\n", result.Created, result.Updated, result.Disabled)
}

func parseOptions() options {
	apply := flag.Bool("apply", false, "将港口数据写入数据库")
	source := flag.String("source", "", "UNECE UN/LOCODE 官方 ZIP 路径")
	release := flag.String("release", "", "数据版本；文件名无版本前缀时必填")
	organizationCode := flag.String("org-code", strings.TrimSpace(os.Getenv("BOOTSTRAP_ORGANIZATION_CODE")), "目标组织代码")
	flag.Parse()
	path := strings.TrimSpace(*source)
	code := strings.TrimSpace(*organizationCode)
	if path == "" || code == "" {
		fmt.Fprintln(os.Stderr, "source 和 org-code 均不能为空")
		os.Exit(2)
	}
	return options{apply: *apply, source: filepath.Clean(path), release: strings.TrimSpace(*release), organizationCode: code}
}

func readCodeListFiles(raw []byte) (map[string][]byte, string, error) {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, "", err
	}
	wanted := make(map[string]struct{}, len(partNames))
	for _, name := range partNames {
		wanted[name] = struct{}{}
	}
	files := make(map[string][]byte, len(partNames))
	releases := make(map[string]struct{})
	for _, entry := range reader.File {
		base := filepath.Base(strings.ReplaceAll(entry.Name, "\\", "/"))
		matches := releasePattern.FindStringSubmatch(base)
		normalized := releasePattern.ReplaceAllString(base, "")
		if _, exists := wanted[normalized]; !exists {
			continue
		}
		if _, duplicate := files[normalized]; duplicate {
			return nil, "", fmt.Errorf("发布包包含重复文件 %q", normalized)
		}
		content, readErr := readZipEntry(entry)
		if readErr != nil {
			return nil, "", fmt.Errorf("读取 %s: %w", entry.Name, readErr)
		}
		files[normalized] = content
		if len(matches) == 2 {
			releases[matches[1]] = struct{}{}
		}
	}
	for _, name := range partNames {
		if _, exists := files[name]; !exists {
			return nil, "", fmt.Errorf("发布包缺少 %q", name)
		}
	}
	if len(releases) > 1 {
		return nil, "", fmt.Errorf("发布包包含多个版本前缀")
	}
	for release := range releases {
		return files, release, nil
	}
	return files, "", nil
}

func readZipEntry(entry *zip.File) ([]byte, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, 64<<20))
}

func parseUNLocode(files map[string][]byte, zipRaw []byte) ([]data.PortSyncRecord, unlocodeParseSummary, error) {
	summary := unlocodeParseSummary{SourceHash: fmt.Sprintf("%x", sha256.Sum256(zipRaw))}
	type locationRow struct {
		fileName string
		line     int
		record   []string
		priority int
	}
	locationsByCode := make(map[string]locationRow)
	for _, name := range partNames {
		text, err := decodeCSV(files[name])
		if err != nil {
			return nil, summary, fmt.Errorf("解码 %s: %w", name, err)
		}
		reader := csv.NewReader(strings.NewReader(text))
		reader.FieldsPerRecord = -1
		for line := 1; ; line++ {
			record, readErr := reader.Read()
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return nil, summary, fmt.Errorf("读取 %s 第 %d 行: %w", name, line, readErr)
			}
			summary.RawRows++
			if len(record) < 12 {
				summary.InvalidRows++
				continue
			}
			change := strings.TrimSpace(record[0])
			country := strings.ToUpper(strings.TrimSpace(record[1]))
			location := strings.ToUpper(strings.TrimSpace(record[2]))
			nameEN := strings.TrimSpace(record[3])
			if location == "" && strings.HasPrefix(nameEN, ".") {
				summary.CountryHeaders++
				continue
			}
			if change == "=" {
				summary.Aliases++
				continue
			}
			code := country + location
			if !countryPattern.MatchString(country) || !unlocodePattern.MatchString(code) {
				summary.InvalidRows++
				continue
			}
			priority := changePriority(change)
			if existing, exists := locationsByCode[code]; exists {
				if priority == existing.priority {
					if isPortFunction(record) || isPortFunction(existing.record) {
						summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("%s 第 %d 行 UN/LOCODE %s 与 %s 第 %d 行同优先级重复", name, line, code, existing.fileName, existing.line))
					}
					continue
				}
				summary.Superseded++
				if priority < existing.priority {
					continue
				}
			}
			locationsByCode[code] = locationRow{fileName: name, line: line, record: append([]string(nil), record...), priority: priority}
		}
	}
	rowsByCode := make(map[string]data.PortSyncRecord)
	for code, location := range locationsByCode {
		record := location.record
		functionCode := strings.TrimSpace(record[6])
		if len(functionCode) == 0 || functionCode[0] != '1' {
			summary.NonPorts++
			continue
		}
		country := strings.ToUpper(strings.TrimSpace(record[1]))
		nameEN := strings.TrimSpace(record[3])
		if nameEN == "" || len([]rune(nameEN)) > 200 {
			summary.InvalidRows++
			summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("%s 第 %d 行港口 %s 名称不符合约束", location.fileName, location.line, code))
			continue
		}
		modes := []string{"SEA"}
		if len(functionCode) > 1 && functionCode[1] == '2' {
			modes = append(modes, "RAIL")
		}
		if len(functionCode) > 2 && functionCode[2] == '3' {
			modes = append(modes, "ROAD")
		}
		enabled := strings.ToUpper(strings.TrimSpace(record[7])) != "XX"
		if !enabled {
			summary.Withdrawn++
		}
		if _, exists := rowsByCode[code]; exists {
			summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("有效港口 UN/LOCODE %s 重复", code))
			continue
		}
		rowsByCode[code] = data.PortSyncRecord{UNLocode: code, NameEN: nameEN, CountryCode: country, TransportModes: modes, Enabled: enabled}
	}
	if len(rowsByCode) == 0 {
		return nil, summary, errors.New("未解析到有效海港记录")
	}
	rows := make([]data.PortSyncRecord, 0, len(rowsByCode))
	for _, row := range rowsByCode {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].UNLocode < rows[j].UNLocode })
	summary.ValidPorts = len(rows)
	return rows, summary, nil
}

func changePriority(change string) int {
	switch change {
	case "#":
		return 3
	case "+":
		return 2
	case "|", "¦":
		return 1
	default:
		return 0
	}
}

func isPortFunction(record []string) bool {
	if len(record) <= 6 {
		return false
	}
	functionCode := strings.TrimSpace(record[6])
	return len(functionCode) > 0 && functionCode[0] == '1'
}

func decodeCSV(raw []byte) (string, error) {
	if utf8.Valid(raw) {
		return string(raw), nil
	}
	decoded, err := charmap.Windows1252.NewDecoder().Bytes(raw)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func openStore() (*data.IndustryReferenceSyncStore, func(), error) {
	databaseSource := strings.TrimSpace(os.Getenv("DATABASE_SOURCE"))
	if databaseSource == "" {
		return nil, nil, errors.New("DATABASE_SOURCE 不能为空")
	}
	storage, cleanup, err := data.NewData(&conf.Data{Database: &conf.Data_Database{Driver: "postgres", Source: databaseSource}})
	if err != nil {
		return nil, nil, err
	}
	return data.NewIndustryReferenceSyncStore(storage), cleanup, nil
}

func printSummary(options options, summary unlocodeParseSummary, conflicts []data.IndustryReferenceSyncConflict) {
	fmt.Printf("UN/LOCODE 数据源：%s\n", options.source)
	fmt.Printf("组织：%s，版本：%s，SHA-256：%s\n", options.organizationCode, options.release, summary.SourceHash)
	fmt.Printf("原始行 %d，有效海港 %d，非海港 %d，国家标题 %d，别名 %d，变更覆盖 %d，无效行 %d，撤销港口 %d\n", summary.RawRows, summary.ValidPorts, summary.NonPorts, summary.CountryHeaders, summary.Aliases, summary.Superseded, summary.InvalidRows, summary.Withdrawn)
	for index, problem := range summary.FatalProblems {
		if index == 10 {
			fmt.Printf("其余源文件冲突 %d 条未展开\n", len(summary.FatalProblems)-index)
			break
		}
		fmt.Printf("源文件冲突：%s\n", problem)
	}
	for index, conflict := range conflicts {
		if index == 10 {
			fmt.Printf("其余数据库冲突 %d 条未展开\n", len(conflicts)-index)
			break
		}
		fmt.Printf("数据库冲突：%s %s\n", conflict.Code, conflict.Message)
	}
}

func fail(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	os.Exit(1)
}
