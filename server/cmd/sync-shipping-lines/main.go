package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/roncin/roncin-go-admin/server/cmd/internal/syncrunner"
	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/xuri/excelize/v2"
)

const smdgSource = "SMDG_LCL"

type shippingLineParseSummary struct {
	RawRows       int
	ValidLiners   int
	SkippedEmpty  int
	InvalidCodes  int
	FatalProblems []string
	SourceHash    string
}

func main() {
	syncrunner.Run(run)
}

func run(ctx context.Context) error {
	options := parseOptions()
	raw, err := os.ReadFile(options.Source)
	if err != nil {
		return fmt.Errorf("读取 SMDG 船公司数据文件失败: %w", err)
	}
	hash := sha256.Sum256(raw)
	sourceHash := hex.EncodeToString(hash[:])

	rows, detectedRelease, summary, err := parseShippingLines(options.Source, sourceHash)
	if err != nil {
		return fmt.Errorf("解析 SMDG 船公司数据失败: %w", err)
	}
	if options.Release == "" {
		options.Release = detectedRelease
	}
	if options.Release == "" {
		options.Release = "SMDG-LCL"
	}

	store, cleanup, err := syncrunner.OpenStore()
	if err != nil {
		return fmt.Errorf("初始化船公司同步存储失败: %w", err)
	}
	defer cleanup()

	conflicts, err := store.CheckShippingLines(ctx, smdgSource, rows)
	if err != nil {
		return fmt.Errorf("检查船公司同步冲突失败: %w", err)
	}

	printSummary(options.Source, options, summary, conflicts)
	if len(summary.FatalProblems) > 0 || len(conflicts) > 0 {
		return fmt.Errorf("船公司数据存在冲突: 源文件冲突 %d 条，数据库冲突 %d 条", len(summary.FatalProblems), len(conflicts))
	}

	if !options.Apply {
		fmt.Println("当前为预览模式；确认统计后使用 -apply 写入数据库")
		return nil
	}

	result, err := store.ApplyShippingLines(ctx, smdgSource, rows)
	if err != nil {
		return fmt.Errorf("写入船公司数据失败: %w", err)
	}
	fmt.Printf("船公司同步完成：新增 %d，更新 %d，停用 %d\n", result.Created, result.Updated, result.Disabled)
	return nil
}

func parseOptions() syncrunner.Options {
	apply := flag.Bool("apply", false, "将船公司数据写入数据库")
	source := flag.String("source", "", "SMDG Liner codes list (.xlsx) 路径")
	release := flag.String("release", "", "数据版本；默认从 Sheet 名或文件名识别")
	flag.Parse()

	path := resolveSourcePath(strings.TrimSpace(*source))
	if path == "" {
		// 检查默认可能路径
		defaultPaths := []string{
			filepath.Join("seeds", "SMDG_Liner-codes-list-20260903.xlsx"),
			filepath.Join("server", "seeds", "SMDG_Liner-codes-list-20260903.xlsx"),
			filepath.Join("..", "seeds", "SMDG_Liner-codes-list-20260903.xlsx"),
			"/tmp/dinotty/SMDG_Liner-codes-list-20260903.xlsx",
			filepath.Join("..", ".cache", "master-data", "SMDG_Liner-codes-list.xlsx"),
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "source 不能为空，未找到默认的 SMDG Excel 文件")
		os.Exit(2)
	}
	return syncrunner.Options{Apply: *apply, Source: filepath.Clean(path), Release: strings.TrimSpace(*release)}
}

func resolveSourcePath(source string) string {
	if source == "" || filepath.IsAbs(source) {
		return source
	}
	initialDirectory := strings.TrimSpace(os.Getenv("INIT_CWD"))
	if initialDirectory == "" {
		return filepath.Clean(source)
	}
	return filepath.Clean(filepath.Join(initialDirectory, source))
}

func parseShippingLines(filePath, sourceHash string) ([]data.ShippingLineSyncRecord, string, shippingLineParseSummary, error) {
	summary := shippingLineParseSummary{SourceHash: sourceHash}
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, "", summary, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, "", summary, errors.New("工作簿不包含任何工作表")
	}

	targetSheet := sheets[0]
	detectedRelease := targetSheet

	allRows, err := f.GetRows(targetSheet)
	if err != nil {
		return nil, "", summary, fmt.Errorf("读取工作表 %s 失败: %w", targetSheet, err)
	}

	summary.RawRows = len(allRows)
	// 第 9 行（索引 8）是表头，数据从第 10 行（索引 9）开始
	if len(allRows) < 10 {
		return nil, "", summary, errors.New("表格行数不足，未找到有效数据行")
	}

	recordsByCode := make(map[string]data.ShippingLineSyncRecord)
	for i := 9; i < len(allRows); i++ {
		row := allRows[i]
		if len(row) == 0 {
			summary.SkippedEmpty++
			continue
		}
		code := strings.ToUpper(strings.TrimSpace(cell(row, 0)))
		if code == "" {
			summary.SkippedEmpty++
			continue
		}
		if !scacPattern.MatchString(code) {
			summary.InvalidCodes++
			continue
		}
		nameEN := strings.TrimSpace(cell(row, 1))
		if nameEN == "" {
			summary.SkippedEmpty++
			continue
		}
		website := strings.TrimSpace(cell(row, 8))
		address := strings.TrimSpace(cell(row, 9))

		country := resolveCountryCode(code, nameEN, address, website)
		nameZH := resolveChineseName(code, nameEN)
		alliance := resolveAlliance(code)
		prefixes := resolveContainerPrefixes(code)
		trackingURL := resolveTrackingURL(website)

		if existing, exists := recordsByCode[code]; exists {
			summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("第 %d 行船公司代码 %s 与已有代码冲突 (%s vs %s)", i+1, code, nameEN, existing.NameEN))
			continue
		}

		recordsByCode[code] = data.ShippingLineSyncRecord{
			SCACCode:          code,
			NameZH:            nameZH,
			NameEN:            nameEN,
			CountryCode:       country,
			TrackingURL:       trackingURL,
			Alliance:          alliance,
			ContainerPrefixes: prefixes,
			Enabled:           true,
		}
	}

	rows := make([]data.ShippingLineSyncRecord, 0, len(recordsByCode))
	for _, row := range recordsByCode {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].SCACCode < rows[j].SCACCode })
	summary.ValidLiners = len(rows)

	return rows, detectedRelease, summary, nil
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func printSummary(sourcePath string, options syncrunner.Options, summary shippingLineParseSummary, conflicts []data.IndustryReferenceSyncConflict) {
	fmt.Printf("船公司数据源：%s\n", sourcePath)
	fmt.Printf("版本：%s，SHA-256：%s\n", options.Release, summary.SourceHash)
	fmt.Printf("原始行 %d，有效船公司 %d，空行跳过 %d，非法代码跳过 %d\n", summary.RawRows, summary.ValidLiners, summary.SkippedEmpty, summary.InvalidCodes)
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
