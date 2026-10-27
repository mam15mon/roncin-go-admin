package main

import (
	"context"
	"crypto/sha256"
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
	RawRows         int
	ValidLiners     int
	SkippedEmpty    int
	InvalidCodes    int
	FatalProblems   []string
	SourceHash      string
	ChineseHash     string
	ChineseNames    int
	ExistingNames   int
	TranslatedNames int
}

func main() {
	syncrunner.Run(run)
}

type shippingLineOptions struct {
	syncrunner.Options
	ChineseSource string
}

func run(ctx context.Context) error {
	return runWithOptions(ctx, parseOptions())
}

func runWithOptions(ctx context.Context, options shippingLineOptions) error {
	raw, err := os.ReadFile(options.Source)
	if err != nil {
		return fmt.Errorf("读取 SMDG 船公司数据文件失败: %w", err)
	}
	sourceHash := fmt.Sprintf("%x", sha256.Sum256(raw))

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
	if len(summary.FatalProblems) > 0 {
		printSummary(options.Source, options, summary, nil)
		return fmt.Errorf("船公司源文件校验失败: %d 条问题", len(summary.FatalProblems))
	}

	chineseRaw, err := os.ReadFile(options.ChineseSource)
	if err != nil {
		return fmt.Errorf("读取船公司中文对照失败: %w", err)
	}
	rows, err = attachChineseNames(rows, chineseRaw, &summary)
	if err != nil {
		return fmt.Errorf("校验船公司中文对照失败: %w", err)
	}
	if !options.Apply && strings.TrimSpace(os.Getenv("DATABASE_SOURCE")) == "" {
		printSummary(options.Source, options, summary, nil)
		fmt.Println("当前为离线预览模式，未检查数据库冲突；配置 DATABASE_SOURCE 后可预检，使用 -apply 显式写入")
		return nil
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
	if len(conflicts) > 0 {
		return fmt.Errorf("船公司数据存在 %d 条数据库冲突", len(conflicts))
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

func parseOptions() shippingLineOptions {
	apply := flag.Bool("apply", false, "将船公司数据写入数据库")
	source := flag.String("source", "", "SMDG Liner codes list (.xlsx) 路径")
	chineseSource := flag.String("chinese-source", "", "配套船公司中文 CSV 路径；默认读取 Excel 同目录 shipping-lines.zh-CN.csv")
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
	chinesePath := resolveSourcePath(strings.TrimSpace(*chineseSource))
	if chinesePath == "" {
		chinesePath = filepath.Join(filepath.Dir(path), "shipping-lines.zh-CN.csv")
	}
	return shippingLineOptions{Options: syncrunner.Options{Apply: *apply, Source: filepath.Clean(path), Release: strings.TrimSpace(*release)}, ChineseSource: chinesePath}
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
		if !smdgLinerCodePattern.MatchString(code) {
			summary.InvalidCodes++
			summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("第 %d 行船公司代码 %q 不是三位字母或数字", i+1, code))
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
		alliance := resolveAlliance(code)
		prefixes := resolveContainerPrefixes(code)
		for _, prefix := range prefixes {
			if !prefixPattern.MatchString(prefix) {
				summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("船公司 %s 箱号前缀 %q 不符合三位字母加 U/J/Z 的格式", code, prefix))
			}
		}
		trackingURL := resolveTrackingURL(website)

		if existing, exists := recordsByCode[code]; exists {
			summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("第 %d 行船公司代码 %s 与已有代码冲突 (%s vs %s)", i+1, code, nameEN, existing.NameEN))
			continue
		}

		recordsByCode[code] = data.ShippingLineSyncRecord{
			SCACCode:          code,
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

func printSummary(sourcePath string, options shippingLineOptions, summary shippingLineParseSummary, conflicts []data.IndustryReferenceSyncConflict) {
	fmt.Printf("船公司数据源：%s\n", sourcePath)
	fmt.Printf("版本：%s，SHA-256：%s\n", options.Release, summary.SourceHash)
	fmt.Printf("原始行 %d，有效船公司 %d，空行跳过 %d，非法代码 %d\n", summary.RawRows, summary.ValidLiners, summary.SkippedEmpty, summary.InvalidCodes)
	fmt.Printf("中文对照：%s，SHA-256：%s\n", options.ChineseSource, summary.ChineseHash)
	fmt.Printf("中文覆盖 %d/%d，已有名称 %d，编译名称 %d\n", summary.ChineseNames, summary.ValidLiners, summary.ExistingNames, summary.TranslatedNames)
	for index, problem := range summary.FatalProblems {
		if index == 10 {
			fmt.Printf("其余源文件问题 %d 条未展开\n", len(summary.FatalProblems)-index)
			break
		}
		fmt.Printf("源文件问题：%s\n", problem)
	}
	for index, conflict := range conflicts {
		if index == 10 {
			fmt.Printf("其余数据库冲突 %d 条未展开\n", len(conflicts)-index)
			break
		}
		fmt.Printf("数据库冲突：%s %s\n", conflict.Code, conflict.Message)
	}
}
