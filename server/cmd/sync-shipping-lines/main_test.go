package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestDictResolutions(t *testing.T) {
	t.Parallel()

	// 2. 航运联盟映射测试
	if alliance := resolveAlliance("COS"); alliance == nil || *alliance != "Ocean Alliance" {
		t.Fatalf("COS 联盟 = %v, want Ocean Alliance", alliance)
	}
	if alliance := resolveAlliance("MSK"); alliance == nil || *alliance != "Gemini Cooperation" {
		t.Fatalf("MSK 联盟 = %v, want Gemini Cooperation", alliance)
	}
	if alliance := resolveAlliance("XYZ"); alliance != nil {
		t.Fatalf("未知船司联盟应为空，got %v", *alliance)
	}

	// 3. 集装箱前缀测试
	if prefixes := resolveContainerPrefixes("COS"); len(prefixes) < 2 || prefixes[0] != "COSU" {
		t.Fatalf("COS 箱号前缀 = %v", prefixes)
	}

	// 4. 国家代码识别测试
	if country := resolveCountryCode("OOL", "Orient Overseas Container Line", "", ""); country != "HK" {
		t.Fatalf("OOL 国家代码 = %s, want HK", country)
	}
	if country := resolveCountryCode("COS", "COSCO", "5299 Binjiang Dadao, Shanghai, China", ""); country != "CN" {
		t.Fatalf("COS 国家代码 = %s, want CN", country)
	}
	if country := resolveCountryCode("MSK", "Maersk", "Esplanaden 50, 1098 Copenhagen, Denmark", ""); country != "DK" {
		t.Fatalf("MSK 国家代码 = %s, want DK", country)
	}
	if country := resolveCountryCode("UNKNOWN", "Unknown Line", "", ""); country != "UN" {
		t.Fatalf("未知国家代码保底 = %s, want UN", country)
	}

	// 5. Tracking URL 规范化测试
	if url := resolveTrackingURL("http://www.maersk.com"); url == nil || *url != "http://www.maersk.com" {
		t.Fatalf("http url = %v", url)
	}
	if url := resolveTrackingURL("www.cosco.com"); url == nil || *url != "https://www.cosco.com" {
		t.Fatalf("www url = %v", url)
	}
	if url := resolveTrackingURL("not available"); url != nil {
		t.Fatalf("not available 应为 nil, got %v", *url)
	}
	if url := resolveTrackingURL("-"); url != nil {
		t.Fatalf("- 应为 nil, got %v", *url)
	}
}

func TestParseShippingLinesExcel(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	xlsxPath := filepath.Join(tmpDir, "test_liners.xlsx")

	f := excelize.NewFile()
	sheetName := "20260903"
	f.SetSheetName("Sheet1", sheetName)

	// 模拟 1~8 行元信息与标题
	for i := 1; i <= 8; i++ {
		f.SetCellValue(sheetName, "A"+string(rune('0'+i)), "meta")
	}
	// 第 9 行表头
	headers := []string{"Code", "Line", "Parent", "NVOCC", "VOCC", "Last change", "Valid from", "Valid until", "Website", "Address"}
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 9)
		f.SetCellValue(sheetName, cell, h)
	}

	// 第 10 行：正常 COSCO
	f.SetCellValue(sheetName, "A10", "COS")
	f.SetCellValue(sheetName, "B10", "COSCO SHIPPING Lines Co. Ltd")
	f.SetCellValue(sheetName, "I10", "http://en.coscocs.com/")
	f.SetCellValue(sheetName, "J10", "5299 Binjiang Dadao, Shanghai, China")

	// 第 11 行：正常 Maersk
	f.SetCellValue(sheetName, "A11", "MSK")
	f.SetCellValue(sheetName, "B11", "Maersk Line")
	f.SetCellValue(sheetName, "I11", "www.maersk.com")
	f.SetCellValue(sheetName, "J11", "Copenhagen, Denmark")

	// 第 12 行：SMDG 官方船公司代码可含数字。
	f.SetCellValue(sheetName, "A12", "G2O")
	f.SetCellValue(sheetName, "B12", "G2 Ocean AS")

	f.SetCellValue(sheetName, "A13", "TI2")
	f.SetCellValue(sheetName, "B13", "Ti2 Container lines pte ltd")

	// 真正非法的代码必须记录为源文件问题，禁止静默漏导。
	f.SetCellValue(sheetName, "A14", "G@O")
	f.SetCellValue(sheetName, "B14", "Invalid Line")

	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatalf("保存测试 xlsx 失败: %v", err)
	}
	defer os.Remove(xlsxPath)

	rows, detectedRelease, summary, err := parseShippingLines(xlsxPath, "test-hash")
	if err != nil {
		t.Fatalf("parseShippingLines() error = %v", err)
	}

	if detectedRelease != "20260903" {
		t.Fatalf("detectedRelease = %s, want 20260903", detectedRelease)
	}
	if summary.ValidLiners != 4 {
		t.Fatalf("ValidLiners = %d, want 4", summary.ValidLiners)
	}
	if summary.InvalidCodes != 1 || len(summary.FatalProblems) != 1 || !strings.Contains(summary.FatalProblems[0], "第 14 行") {
		t.Fatalf("非法代码没有定位到源文件行: %+v", summary)
	}
	if len(rows) != 4 {
		t.Fatalf("len(rows) = %d, want 4", len(rows))
	}

	// 验证按代码排序
	if rows[0].SCACCode != "COS" || rows[0].CountryCode != "CN" {
		t.Fatalf("rows[0] = %+v", rows[0])
	}
	if rows[1].SCACCode != "G2O" || rows[1].NameEN != "G2 Ocean AS" {
		t.Fatalf("rows[1] = %+v", rows[1])
	}
	if rows[2].SCACCode != "MSK" || rows[2].CountryCode != "DK" {
		t.Fatalf("rows[2] = %+v", rows[2])
	}
	if rows[3].SCACCode != "TI2" || rows[3].NameEN != "Ti2 Container lines pte ltd" {
		t.Fatalf("rows[3] = %+v", rows[3])
	}
}

func TestParseShippingLinesArchivedSnapshot(t *testing.T) {
	rows, release, summary, err := parseShippingLines(filepath.Join("..", "..", "seeds", "SMDG_Liner-codes-list-20260903.xlsx"), "test-hash")
	if err != nil {
		t.Fatal(err)
	}
	if release != "20260903" || summary.RawRows != 369 || summary.ValidLiners != 360 || summary.SkippedEmpty != 0 || summary.InvalidCodes != 0 || len(summary.FatalProblems) != 0 || len(rows) != 360 {
		t.Fatalf("归档船公司快照解析不完整: release=%s summary=%+v rows=%d", release, summary, len(rows))
	}
	for code, name := range map[string]string{"G2O": "G2 Ocean AS", "TI2": "Ti2 Container lines pte ltd"} {
		found := false
		for _, row := range rows {
			if row.SCACCode == code && row.NameEN == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("归档快照缺失船公司 %s / %s", code, name)
		}
	}
}
