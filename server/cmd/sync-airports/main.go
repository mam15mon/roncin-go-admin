package main

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

const (
	ourAirportsURL    = "https://ourairports.com/data/airports.csv"
	ourAirportsSource = "OURAIRPORTS"
	requestTimeout    = 30 * time.Second
	maxDownloadSize   = 100 << 20
)

var (
	iataPattern    = regexp.MustCompile(`^[A-Z]{3}$`)
	icaoPattern    = regexp.MustCompile(`^[A-Z0-9]{4}$`)
	countryPattern = regexp.MustCompile(`^[A-Z]{2}$`)
)

type options struct {
	apply            bool
	source           string
	release          string
	organizationCode string
}

type airportParseSummary struct {
	RawRows       int
	ValidRows     int
	SkippedIATA   int
	InvalidICAO   int
	Closed        int
	SourceHash    string
	FatalProblems []string
}

func main() {
	ctx := context.Background()
	options := parseOptions()
	raw, sourceLabel, err := loadSource(ctx, options.source)
	if err != nil {
		fail("读取 OurAirports 数据失败", err)
	}
	rows, summary, err := parseAirports(raw)
	if err != nil {
		fail("解析 OurAirports 数据失败", err)
	}
	if options.release == "" {
		options.release = time.Now().Format(time.DateOnly)
	}
	if options.source == "" {
		cached, cacheErr := cacheDownload(raw, options.release, summary.SourceHash)
		if cacheErr != nil {
			fail("缓存 OurAirports 数据失败", cacheErr)
		}
		sourceLabel = cached
	}

	store, cleanup, err := openStore()
	if err != nil {
		fail("初始化机场同步存储失败", err)
	}
	defer cleanup()
	conflicts, err := store.CheckAirports(ctx, options.organizationCode, ourAirportsSource, rows)
	if err != nil {
		fail("检查机场同步冲突失败", err)
	}
	printSummary(sourceLabel, options, summary, conflicts)
	if len(summary.FatalProblems) > 0 || len(conflicts) > 0 {
		fail("机场数据存在致命冲突", fmt.Errorf("源文件冲突 %d 条，数据库冲突 %d 条", len(summary.FatalProblems), len(conflicts)))
	}
	if !options.apply {
		fmt.Println("当前为预览模式；确认统计后使用 -apply 写入数据库")
		return
	}
	result, err := store.ApplyAirports(ctx, options.organizationCode, ourAirportsSource, options.release, summary.SourceHash, rows)
	if err != nil {
		fail("写入机场数据失败", err)
	}
	fmt.Printf("机场同步完成：新增 %d，更新 %d，停用 %d\n", result.Created, result.Updated, result.Disabled)
}

func parseOptions() options {
	apply := flag.Bool("apply", false, "将机场数据写入数据库")
	source := flag.String("source", "", "本地 airports.csv 路径；为空时从 OurAirports 下载")
	release := flag.String("release", "", "数据版本；默认使用同步日期")
	organizationCode := flag.String("org-code", strings.TrimSpace(os.Getenv("BOOTSTRAP_ORGANIZATION_CODE")), "目标组织代码")
	flag.Parse()
	code := strings.TrimSpace(*organizationCode)
	if code == "" {
		fmt.Fprintln(os.Stderr, "org-code 不能为空，也未配置 BOOTSTRAP_ORGANIZATION_CODE")
		os.Exit(2)
	}
	return options{apply: *apply, source: strings.TrimSpace(*source), release: strings.TrimSpace(*release), organizationCode: code}
}

func loadSource(ctx context.Context, source string) ([]byte, string, error) {
	if source != "" {
		raw, err := os.ReadFile(source)
		return raw, source, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ourAirportsURL, nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("User-Agent", "roncin-go-admin-airport-sync/1.0")
	client := &http.Client{Timeout: requestTimeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("OurAirports 返回 HTTP %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxDownloadSize+1))
	if err != nil {
		return nil, "", err
	}
	if len(raw) > maxDownloadSize {
		return nil, "", fmt.Errorf("下载内容超过 %d 字节限制", maxDownloadSize)
	}
	return raw, ourAirportsURL, nil
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

func cacheDownload(raw []byte, release, hash string) (string, error) {
	directory := filepath.Join("..", ".cache", "master-data")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(directory, fmt.Sprintf("ourairports-airports-%s-%s.csv", release, hash[:12]))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
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

func printSummary(sourceLabel string, options options, summary airportParseSummary, conflicts []data.IndustryReferenceSyncConflict) {
	fmt.Printf("机场数据源：%s\n", sourceLabel)
	fmt.Printf("组织：%s，版本：%s，SHA-256：%s\n", options.organizationCode, options.release, summary.SourceHash)
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

func fail(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	os.Exit(1)
}
