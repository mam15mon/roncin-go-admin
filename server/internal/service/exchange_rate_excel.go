package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/xuri/excelize/v2"
)

const (
	exchangeRateImportSheet       = "汇率导入"
	exchangeRateImportHelpSheet   = "填写说明"
	exchangeRateImportMaxFileSize = 5 << 20
)

var exchangeRateImportHeaders = []string{"原币", "本币", "应收汇率（现汇卖出价）", "应付汇率（现汇买入价）", "基准汇率", "生效开始时间"}

func buildExchangeRateImportTemplate() ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()
	defaultSheet := file.GetSheetName(0)
	if err := file.SetSheetName(defaultSheet, exchangeRateImportSheet); err != nil {
		return nil, err
	}
	if _, err := file.NewSheet(exchangeRateImportHelpSheet); err != nil {
		return nil, err
	}
	for index, header := range exchangeRateImportHeaders {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		if err := file.SetCellValue(exchangeRateImportSheet, cell, header); err != nil {
			return nil, err
		}
	}
	headerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"1677FF"}, Pattern: 1}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	if err != nil {
		return nil, err
	}
	if err = file.SetCellStyle(exchangeRateImportSheet, "A1", "F1", headerStyle); err != nil {
		return nil, err
	}
	timeFormat := "yyyy-mm-dd hh:mm:ss"
	timeStyle, err := file.NewStyle(&excelize.Style{CustomNumFmt: &timeFormat})
	if err != nil {
		return nil, err
	}
	if err = file.SetCellStyle(exchangeRateImportSheet, "F2", "F501", timeStyle); err != nil {
		return nil, err
	}
	widths := map[string]float64{"A": 12, "B": 12, "C": 22, "D": 22, "E": 16, "F": 24}
	for column, width := range widths {
		if err = file.SetColWidth(exchangeRateImportSheet, column, column, width); err != nil {
			return nil, err
		}
	}
	if err = file.SetPanes(exchangeRateImportSheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return nil, err
	}
	help := [][]any{
		{"模板版本", biz.ExchangeRateImportTemplateVersion},
		{"填写规则", "生效开始时间精确到秒，格式为 YYYY-MM-DD HH:mm:ss，按 Asia/Shanghai 解释；任选目标自然周内时刻，服务端自动归一化为该自然周（周一 00:00:00 至周日 23:59:59）。"},
		{"应收汇率", "现汇卖出价口径：客户账单（AR）外币折本币收款使用，最多 8 位小数且必须大于 0。"},
		{"应付汇率", "现汇买入价口径：供应商账单（AP）外币折本币付款使用，最多 8 位小数且必须大于 0。"},
		{"基准汇率", "中行折算价口径（可空）：内部综合审计与报表基准；留空按应收/应付中间价记录。"},
		{"导入策略", "整批导入：任一行错误或文件内同周重复时不能确认；与现有同周汇率重复时执行覆盖更新。"},
		{"最大行数", biz.ExchangeRateImportMaxRows},
	}
	for rowIndex, row := range help {
		for columnIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			if err = file.SetCellValue(exchangeRateImportHelpSheet, cell, value); err != nil {
				return nil, err
			}
		}
	}
	if err = file.SetColWidth(exchangeRateImportHelpSheet, "A", "A", 16); err != nil {
		return nil, err
	}
	if err = file.SetColWidth(exchangeRateImportHelpSheet, "B", "B", 100); err != nil {
		return nil, err
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func parseExchangeRateImportWorkbook(fileName string, content []byte) (biz.PreviewExchangeRateImportInput, error) {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" || strings.ToLower(filepath.Ext(fileName)) != ".xlsx" || len(content) == 0 || len(content) > exchangeRateImportMaxFileSize {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	file, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{UnzipSizeLimit: 32 << 20, UnzipXMLSizeLimit: 8 << 20})
	if err != nil {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	defer file.Close()
	versionText, err := file.GetCellValue(exchangeRateImportHelpSheet, "B1")
	if err != nil {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	version, err := strconv.Atoi(strings.TrimSpace(versionText))
	if err != nil || version != biz.ExchangeRateImportTemplateVersion {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	iterator, err := file.Rows(exchangeRateImportSheet)
	if err != nil {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	defer iterator.Close()
	rows := make([]*biz.ExchangeRateImportRow, 0)
	rowNumber := 0
	for iterator.Next() {
		rowNumber++
		columns, rowErr := iterator.Columns()
		if rowErr != nil {
			return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
		}
		if rowNumber == 1 {
			if !validExchangeRateImportHeaders(columns) {
				return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
			}
			continue
		}
		if exchangeRateImportRowBlank(columns) {
			continue
		}
		if len(rows) >= biz.ExchangeRateImportMaxRows {
			return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportTooManyRows
		}
		for columnIndex := 1; columnIndex <= len(exchangeRateImportHeaders); columnIndex++ {
			cell, _ := excelize.CoordinatesToCellName(columnIndex, rowNumber)
			formula, formulaErr := file.GetCellFormula(exchangeRateImportSheet, cell)
			if formulaErr != nil || formula != "" {
				return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
			}
		}
		values := make([]string, len(exchangeRateImportHeaders))
		for index := range values {
			if index < len(columns) {
				values[index] = strings.TrimSpace(columns[index])
			}
		}
		rows = append(rows, &biz.ExchangeRateImportRow{RowNumber: rowNumber, FromCurrency: values[0], ToCurrency: values[1], ARRate: values[2], APRate: values[3], Rate: values[4], EffectiveFrom: values[5], Errors: []string{}})
	}
	if err = iterator.Error(); err != nil {
		return biz.PreviewExchangeRateImportInput{}, biz.ErrExchangeRateImportFileInvalid
	}
	checksum := sha256.Sum256(content)
	return biz.PreviewExchangeRateImportInput{FileName: fileName, FileChecksum: hex.EncodeToString(checksum[:]), TemplateVersion: version, Rows: rows}, nil
}

func validExchangeRateImportHeaders(columns []string) bool {
	if len(columns) < len(exchangeRateImportHeaders) {
		return false
	}
	for index, expected := range exchangeRateImportHeaders {
		if strings.TrimSpace(columns[index]) != expected {
			return false
		}
	}
	return true
}

func exchangeRateImportRowBlank(columns []string) bool {
	for index := 0; index < len(columns) && index < len(exchangeRateImportHeaders); index++ {
		if strings.TrimSpace(columns[index]) != "" {
			return false
		}
	}
	return true
}
