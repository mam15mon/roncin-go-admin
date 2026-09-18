package service

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/xuri/excelize/v2"
)

func TestExchangeRateImportTemplateCanBeParsed(t *testing.T) {
	content, err := buildExchangeRateImportTemplate()
	if err != nil {
		t.Fatalf("生成汇率导入模板失败: %v", err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("生成结果不是有效 xlsx: %v", err)
	}
	defer file.Close()
	if version, _ := file.GetCellValue(exchangeRateImportHelpSheet, "B1"); version != "3" {
		t.Fatalf("模板版本错误: %q", version)
	}
}

func TestParseExchangeRateImportWorkbook(t *testing.T) {
	content, err := buildExchangeRateImportTemplate()
	if err != nil {
		t.Fatalf("生成模板失败: %v", err)
	}
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("打开模板失败: %v", err)
	}
	values := []any{"USD", "CNY", "7.30000000", "7.10000000", "7.20000000", "2026-08-27 09:30:01"}
	for index, value := range values {
		cell, _ := excelize.CoordinatesToCellName(index+1, 2)
		if err = file.SetCellValue(exchangeRateImportSheet, cell, value); err != nil {
			t.Fatalf("写测试数据失败: %v", err)
		}
	}
	buffer, err := file.WriteToBuffer()
	_ = file.Close()
	if err != nil {
		t.Fatalf("输出测试工作簿失败: %v", err)
	}
	input, err := parseExchangeRateImportWorkbook("汇率.xlsx", buffer.Bytes())
	if err != nil {
		t.Fatalf("解析汇率工作簿失败: %v", err)
	}
	if input.TemplateVersion != biz.ExchangeRateImportTemplateVersion || len(input.Rows) != 1 || input.Rows[0].RowNumber != 2 || input.Rows[0].EffectiveFrom != "2026-08-27 09:30:01" || input.Rows[0].ARRate != "7.30000000" || input.Rows[0].APRate != "7.10000000" {
		t.Fatalf("工作簿解析结果不正确: %#v", input)
	}
}

func TestParseExchangeRateImportWorkbookRejectsFormula(t *testing.T) {
	content, err := buildExchangeRateImportTemplate()
	if err != nil {
		t.Fatalf("生成模板失败: %v", err)
	}
	file, _ := excelize.OpenReader(bytes.NewReader(content))
	_ = file.SetCellFormula(exchangeRateImportSheet, "C2", "=1+1")
	_ = file.SetCellValue(exchangeRateImportSheet, "A2", "USD")
	buffer, _ := file.WriteToBuffer()
	_ = file.Close()
	if _, err = parseExchangeRateImportWorkbook("汇率.xlsx", buffer.Bytes()); err != biz.ErrExchangeRateImportFileInvalid {
		t.Fatalf("含公式的汇率文件应被拒绝，实际错误为 %v", err)
	}
}

func TestParseExchangeRateImportWorkbookFormatsExcelDateTimeCells(t *testing.T) {
	content, _ := buildExchangeRateImportTemplate()
	file, _ := excelize.OpenReader(bytes.NewReader(content))
	values := []any{"USD", "CNY", "7.2"}
	for index, value := range values {
		cell, _ := excelize.CoordinatesToCellName(index+1, 2)
		_ = file.SetCellValue(exchangeRateImportSheet, cell, value)
	}
	// 生效时刻列（F）由 Excel 日期时间单元格写入，应被格式化为秒级文本。
	_ = file.SetCellValue(exchangeRateImportSheet, "F2", time.Date(2026, 8, 27, 9, 30, 1, 0, time.Local))
	buffer, _ := file.WriteToBuffer()
	_ = file.Close()
	input, err := parseExchangeRateImportWorkbook("汇率.xlsx", buffer.Bytes())
	if err != nil {
		t.Fatalf("Excel 日期时间单元格应能解析: %v", err)
	}
	if input.Rows[0].EffectiveFrom != "2026-08-27 09:30:01" {
		t.Fatalf("Excel 日期时间格式化错误: %#v", input.Rows[0])
	}
}

func TestExcelizeRejectsNegativeSharedStringIndex(t *testing.T) {
	content := buildNegativeSharedStringWorkbook(t)
	testCases := []struct {
		name    string
		options excelize.Options
	}{
		{name: "内存路径"},
		{
			name: "临时文件路径",
			options: excelize.Options{
				UnzipSizeLimit:    4 << 20,
				UnzipXMLSizeLimit: 1 << 10,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			file, err := excelize.OpenReader(bytes.NewReader(content), testCase.options)
			if err != nil {
				t.Fatalf("打开恶意工作簿失败: %v", err)
			}
			defer file.Close()
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("负共享字符串索引不应导致 panic: %v", recovered)
				}
			}()

			if _, err = file.GetCellValue("Sheet1", "A1"); err == nil {
				t.Fatal("负共享字符串索引应返回错误")
			}
			// GetRows 在当前契约中可以把非法索引当作普通值返回，
			// 安全底线是内存与临时文件两条路径都不能 panic。
			_, _ = file.GetRows("Sheet1")
		})
	}
}

func buildNegativeSharedStringWorkbook(t *testing.T) []byte {
	t.Helper()
	file := excelize.NewFile()
	if err := file.SetCellValue("Sheet1", "A1", strings.Repeat("x", 2048)); err != nil {
		t.Fatalf("写入测试工作簿失败: %v", err)
	}
	buffer, err := file.WriteToBuffer()
	_ = file.Close()
	if err != nil {
		t.Fatalf("生成测试工作簿失败: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatalf("读取测试工作簿压缩包失败: %v", err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	modified := false
	for _, zippedFile := range reader.File {
		entryReader, openErr := zippedFile.Open()
		if openErr != nil {
			t.Fatalf("打开工作簿条目 %s 失败: %v", zippedFile.Name, openErr)
		}
		entry, readErr := io.ReadAll(entryReader)
		_ = entryReader.Close()
		if readErr != nil {
			t.Fatalf("读取工作簿条目 %s 失败: %v", zippedFile.Name, readErr)
		}
		if zippedFile.Name == "xl/worksheets/sheet1.xml" {
			updated := bytes.Replace(entry, []byte("<v>0</v>"), []byte("<v>-1</v>"), 1)
			modified = !bytes.Equal(updated, entry)
			entry = updated
		}
		entryWriter, createErr := writer.CreateHeader(&zippedFile.FileHeader)
		if createErr != nil {
			t.Fatalf("创建工作簿条目 %s 失败: %v", zippedFile.Name, createErr)
		}
		if _, writeErr := entryWriter.Write(entry); writeErr != nil {
			t.Fatalf("写入工作簿条目 %s 失败: %v", zippedFile.Name, writeErr)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("完成恶意工作簿失败: %v", err)
	}
	if !modified {
		t.Fatal("未找到待替换的共享字符串索引")
	}
	return output.Bytes()
}
