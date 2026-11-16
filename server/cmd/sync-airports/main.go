package main

import (
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

	"github.com/roncin/roncin-go-admin/server/cmd/internal/syncrunner"
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

const (
	ourAirportsSource = "OURAIRPORTS"
	archivedRelease   = "2026-10-27"
)

var (
	iataPattern    = regexp.MustCompile(`^[A-Z]{3}$`)
	icaoPattern    = regexp.MustCompile(`^[A-Z0-9]{4}$`)
	countryPattern = regexp.MustCompile(`^[A-Z]{2}$`)
)

type airportParseSummary struct {
	RawRows         int
	ValidRows       int
	SkippedIATA     int
	InvalidICAO     int
	Closed          int
	SourceHash      string
	AirportsHash    string
	ChineseHash     string
	ChineseNames    int
	ExistingNames   int
	TranslatedNames int
	FatalProblems   []string
}

func main() {
	syncrunner.Run(run)
}

func run(ctx context.Context) error {
	options, err := parseOptions(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	return runWithOptions(ctx, options)
}

type airportOptions struct {
	syncrunner.Options
	ChineseSource string
}

func runWithOptions(ctx context.Context, options airportOptions) error {
	raw, sourceLabel, err := loadSource(ctx, options.Source)
	if err != nil {
		return fmt.Errorf("读取 OurAirports 数据失败: %w", err)
	}
	rows, summary, err := parseAirports(raw)
	if err != nil {
		return fmt.Errorf("解析 OurAirports 数据失败: %w", err)
	}
	chineseRaw, err := os.ReadFile(options.ChineseSource)
	if err != nil {
		return fmt.Errorf("读取机场中文对照失败: %w", err)
	}
	rows, err = attachChineseNames(rows, chineseRaw, raw, &summary)
	if err != nil {
		return fmt.Errorf("校验机场中文对照失败: %w", err)
	}
	if options.Release == "" {
		options.Release = "sha256:" + summary.AirportsHash[:12]
	}
	if len(summary.FatalProblems) > 0 {
		printSummary(sourceLabel, options.Options, summary, nil)
		return fmt.Errorf("机场源文件存在 %d 条致命冲突", len(summary.FatalProblems))
	}
	if !options.Apply && strings.TrimSpace(os.Getenv("DATABASE_SOURCE")) == "" {
		printSummary(sourceLabel, options.Options, summary, nil)
		fmt.Println("当前为离线预览模式，未检查数据库冲突；配置 DATABASE_SOURCE 后可预检，使用 -apply 显式写入")
		return nil
	}

	store, cleanup, err := syncrunner.OpenStore()
	if err != nil {
		return fmt.Errorf("初始化机场同步存储失败: %w", err)
	}
	defer cleanup()
	conflicts, err := store.CheckAirports(ctx, ourAirportsSource, rows)
	if err != nil {
		return fmt.Errorf("检查机场同步冲突失败: %w", err)
	}
	printSummary(sourceLabel, options.Options, summary, conflicts)
	if len(summary.FatalProblems) > 0 || len(conflicts) > 0 {
		return fmt.Errorf("机场数据存在致命冲突: 源文件冲突 %d 条，数据库冲突 %d 条", len(summary.FatalProblems), len(conflicts))
	}
	if !options.Apply {
		fmt.Println("当前为预览模式；确认统计后使用 -apply 写入数据库")
		return nil
	}
	result, err := store.ApplyAirports(ctx, ourAirportsSource, options.Release, summary.SourceHash, rows)
	if err != nil {
		return fmt.Errorf("写入机场数据失败: %w", err)
	}
	fmt.Printf("机场同步完成：新增 %d，更新 %d，停用 %d\n", result.Created, result.Updated, result.Disabled)
	return nil
}

func parseOptions(args []string) (airportOptions, error) {
	flags := flag.NewFlagSet("sync-airports", flag.ContinueOnError)
	apply := flags.Bool("apply", false, "将机场数据写入数据库")
	source := flags.String("source", "", "本地 airports.csv 路径；默认读取仓库归档快照")
	chineseSource := flags.String("chinese-source", "", "配套中文 CSV 路径；默认读取底表同目录 airports.zh-CN.csv")
	release := flags.String("release", "", "数据版本；默认归档版本，自定义底表未指定时使用内容摘要")
	if err := flags.Parse(args); err != nil {
		return airportOptions{}, err
	}
	if flags.NArg() != 0 {
		return airportOptions{}, fmt.Errorf("不支持的位置参数: %s", strings.Join(flags.Args(), " "))
	}
	path := strings.TrimSpace(*source)
	version := strings.TrimSpace(*release)
	if path == "" {
		for _, candidate := range []string{filepath.Join("seeds", "airports.csv"), filepath.Join("server", "seeds", "airports.csv"), filepath.Join("..", "..", "seeds", "airports.csv")} {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
		if path == "" {
			return airportOptions{}, errors.New("未找到仓库 airports.csv 快照，请通过 -source 指定配套底表")
		}
		if version == "" {
			version = archivedRelease
		}
	}
	chinesePath := strings.TrimSpace(*chineseSource)
	if chinesePath == "" {
		chinesePath = filepath.Join(filepath.Dir(path), "airports.zh-CN.csv")
	}
	return airportOptions{Options: syncrunner.Options{Apply: *apply, Source: path, Release: version}, ChineseSource: chinesePath}, nil
}

func loadSource(_ context.Context, source string) ([]byte, string, error) {
	raw, err := os.ReadFile(source)
	return raw, source, err
}

func parseAirports(raw []byte) ([]data.AirportSyncRecord, airportParseSummary, error) {
	summary := airportParseSummary{SourceHash: fmt.Sprintf("%x", sha256.Sum256(raw))}
	reader := csv.NewReader(strings.NewReader(string(raw)))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, summary, fmt.Errorf("读取 CSV 表头: %w", err)
	}
	indexes, err := requiredIndexes(header, "iata_code", "gps_code", "name", "municipality", "iso_country", "type")
	if err != nil {
		return nil, summary, err
	}
	rowsByIATA := make(map[string]data.AirportSyncRecord)
	icaoOwners := make(map[string]string)
	for line := 2; ; line++ {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, summary, fmt.Errorf("读取 CSV 第 %d 行: %w", line, readErr)
		}
		summary.RawRows++
		iata := strings.ToUpper(strings.TrimSpace(cell(record, indexes["iata_code"])))
		if !iataPattern.MatchString(iata) {
			summary.SkippedIATA++
			continue
		}
		name := strings.TrimSpace(cell(record, indexes["name"]))
		country := strings.ToUpper(strings.TrimSpace(cell(record, indexes["iso_country"])))
		city := strings.TrimSpace(cell(record, indexes["municipality"]))
		if name == "" || len([]rune(name)) > 200 || !countryPattern.MatchString(country) || len([]rune(city)) > 100 {
			summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("第 %d 行 %s 字段不符合长度或国家码约束", line, iata))
			continue
		}
		if _, exists := rowsByIATA[iata]; exists {
			summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("第 %d 行 IATA %s 重复", line, iata))
			continue
		}
		icaoText := strings.ToUpper(strings.TrimSpace(cell(record, indexes["gps_code"])))
		var icao *string
		if icaoText != "" {
			if !icaoPattern.MatchString(icaoText) {
				summary.InvalidICAO++
			} else if owner, exists := icaoOwners[icaoText]; exists {
				summary.FatalProblems = append(summary.FatalProblems, fmt.Sprintf("第 %d 行 ICAO %s 与 IATA %s 重复", line, icaoText, owner))
				continue
			} else {
				icaoOwners[icaoText] = iata
				icao = &icaoText
			}
		}
		var cityPointer *string
		if city != "" {
			cityPointer = &city
		}
		enabled := strings.ToLower(strings.TrimSpace(cell(record, indexes["type"]))) != "closed"
		if !enabled {
			summary.Closed++
		}
		rowsByIATA[iata] = data.AirportSyncRecord{IATACode: iata, ICAOCode: icao, NameEN: name, CityNameEN: cityPointer, CountryCode: country, Enabled: enabled}
	}
	if len(rowsByIATA) == 0 {
		return nil, summary, errors.New("未解析到有效机场记录")
	}
	rows := make([]data.AirportSyncRecord, 0, len(rowsByIATA))
	for _, row := range rowsByIATA {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].IATACode < rows[j].IATACode })
	summary.ValidRows = len(rows)
	return rows, summary, nil
}

func requiredIndexes(header []string, names ...string) (map[string]int, error) {
	indexes := make(map[string]int, len(header))
	for index, name := range header {
		indexes[strings.TrimPrefix(strings.TrimSpace(name), "\ufeff")] = index
	}
	for _, name := range names {
		if _, exists := indexes[name]; !exists {
			return nil, fmt.Errorf("CSV 缺少必需列 %q", name)
		}
	}
	return indexes, nil
}

func cell(record []string, index int) string {
	if index < 0 || index >= len(record) {
		return ""
	}
	return record[index]
}

func printSummary(sourceLabel string, options syncrunner.Options, summary airportParseSummary, conflicts []data.IndustryReferenceSyncConflict) {
	fmt.Printf("机场数据源：%s\n", sourceLabel)
	fmt.Printf("版本：%s，SHA-256：%s\n", options.Release, summary.SourceHash)
	fmt.Printf("机场底表 SHA-256：%s；中文表 SHA-256：%s\n", summary.AirportsHash, summary.ChineseHash)
	fmt.Printf("中文覆盖 %d/%d（已有名称 %d，编制译名 %d）\n", summary.ChineseNames, summary.ValidRows, summary.ExistingNames, summary.TranslatedNames)
	fmt.Printf("原始行 %d，有效机场 %d，无有效 IATA 跳过 %d，非法 ICAO %d，关闭机场 %d\n", summary.RawRows, summary.ValidRows, summary.SkippedIATA, summary.InvalidICAO, summary.Closed)
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
