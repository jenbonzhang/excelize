// Copyright 2016 - 2022 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to and
// read from XLAM / XLSM / XLSX / XLTM / XLTX files. Supports reading and
// writing spreadsheet documents generated by Microsoft Excel™ 2007 and later.
// Supports complex components by high compatibility, and provided streaming
// API for generating or reading data from a worksheet with huge amounts of
// data. This library needs Go version 1.15 or later.

package excelize

import (
	"encoding/xml"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// CellType is the type of cell value type.
type CellType byte

// Cell value types enumeration.
const (
	CellTypeUnset CellType = iota
	CellTypeBool
	CellTypeDate
	CellTypeError
	CellTypeNumber
	CellTypeString
)

const (
	// STCellFormulaTypeArray defined the formula is an array formula.
	STCellFormulaTypeArray = "array"
	// STCellFormulaTypeDataTable defined the formula is a data table formula.
	STCellFormulaTypeDataTable = "dataTable"
	// STCellFormulaTypeNormal defined the formula is a regular cell formula.
	STCellFormulaTypeNormal = "normal"
	// STCellFormulaTypeShared defined the formula is part of a shared formula.
	STCellFormulaTypeShared = "shared"
)

// cellTypes mapping the cell's data type and enumeration.
var cellTypes = map[string]CellType{
	"b":         CellTypeBool,
	"d":         CellTypeDate,
	"n":         CellTypeNumber,
	"e":         CellTypeError,
	"s":         CellTypeString,
	"str":       CellTypeString,
	"inlineStr": CellTypeString,
}

// GetCellValue provides a function to get formatted value from cell by given
// worksheet name and axis in spreadsheet file. If it is possible to apply a
// format to the cell value, it will do so, if not then an error will be
// returned, along with the raw value of the cell. All cells' values will be
// the same in a merged range.
func (f *File) GetCellValue(sheet, axis string, opts ...Options) (string, error) {
	return f.getCellStringFunc(sheet, axis, func(x *xlsxWorksheet, c *xlsxC) (string, bool, error) {
		val, err := c.getValueFrom(f, f.sharedStringsReader(), parseOptions(opts...).RawCellValue)
		return val, true, err
	})
}

// GetCellType provides a function to get the cell's data type by given
// worksheet name and axis in spreadsheet file.
func (f *File) GetCellType(sheet, axis string) (CellType, error) {
	var (
		err         error
		cellTypeStr string
		cellType    CellType
	)
	if cellTypeStr, err = f.getCellStringFunc(sheet, axis, func(x *xlsxWorksheet, c *xlsxC) (string, bool, error) {
		return c.T, true, nil
	}); err != nil {
		return CellTypeUnset, err
	}
	cellType = cellTypes[cellTypeStr]
	return cellType, err
}

// SetCellValue provides a function to set the value of a cell. The specified
// coordinates should not be in the first row of the table, a complex number
// can be set with string text. The following shows the supported data
// types:
//
//    int
//    int8
//    int16
//    int32
//    int64
//    uint
//    uint8
//    uint16
//    uint32
//    uint64
//    float32
//    float64
//    string
//    []byte
//    time.Duration
//    time.Time
//    bool
//    nil
//
// Note that default date format is m/d/yy h:mm of time.Time type value. You can
// set numbers format by SetCellStyle() method.
func (f *File) SetCellValue(sheet, axis string, value interface{}) error {
	var err error
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		err = f.setCellIntFunc(sheet, axis, v)
	case float32:
		err = f.SetCellFloat(sheet, axis, float64(v), -1, 32)
	case float64:
		err = f.SetCellFloat(sheet, axis, v, -1, 64)
	case string:
		err = f.SetCellStr(sheet, axis, v)
	case []byte:
		err = f.SetCellStr(sheet, axis, string(v))
	case time.Duration:
		_, d := setCellDuration(v)
		err = f.SetCellDefault(sheet, axis, d)
		if err != nil {
			return err
		}
		err = f.setDefaultTimeStyle(sheet, axis, 21)
	case time.Time:
		err = f.setCellTimeFunc(sheet, axis, v)
	case bool:
		err = f.SetCellBool(sheet, axis, v)
	case nil:
		err = f.SetCellDefault(sheet, axis, "")
	default:
		err = f.SetCellStr(sheet, axis, fmt.Sprint(value))
	}
	return err
}

// String extracts characters from a string item.
func (x xlsxSI) String() string {
	if len(x.R) > 0 {
		var rows strings.Builder
		for _, s := range x.R {
			if s.T != nil {
				rows.WriteString(s.T.Val)
			}
		}
		return bstrUnmarshal(rows.String())
	}
	if x.T != nil {
		return bstrUnmarshal(x.T.Val)
	}
	return ""
}

// hasValue determine if cell non-blank value.
func (c *xlsxC) hasValue() bool {
	return c.S != 0 || c.V != "" || c.F != nil || c.T != ""
}

// setCellIntFunc is a wrapper of SetCellInt.
func (f *File) setCellIntFunc(sheet, axis string, value interface{}) error {
	var err error
	switch v := value.(type) {
	case int:
		err = f.SetCellInt(sheet, axis, v)
	case int8:
		err = f.SetCellInt(sheet, axis, int(v))
	case int16:
		err = f.SetCellInt(sheet, axis, int(v))
	case int32:
		err = f.SetCellInt(sheet, axis, int(v))
	case int64:
		err = f.SetCellInt(sheet, axis, int(v))
	case uint:
		err = f.SetCellInt(sheet, axis, int(v))
	case uint8:
		err = f.SetCellInt(sheet, axis, int(v))
	case uint16:
		err = f.SetCellInt(sheet, axis, int(v))
	case uint32:
		err = f.SetCellInt(sheet, axis, int(v))
	case uint64:
		err = f.SetCellInt(sheet, axis, int(v))
	}
	return err
}

// setCellTimeFunc provides a method to process time type of value for
// SetCellValue.
func (f *File) setCellTimeFunc(sheet, axis string, value time.Time) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	cellData, col, row, err := f.prepareCell(ws, sheet, axis)
	if err != nil {
		return err
	}
	ws.Lock()
	cellData.S = f.prepareCellStyle(ws, col, row, cellData.S)
	ws.Unlock()

	var isNum bool
	cellData.T, cellData.V, isNum, err = setCellTime(value)
	if err != nil {
		return err
	}
	if isNum {
		_ = f.setDefaultTimeStyle(sheet, axis, 22)
	}
	return err
}

// setCellTime prepares cell type and Excel time by given Go time.Time type
// timestamp.
func setCellTime(value time.Time) (t string, b string, isNum bool, err error) {
	var excelTime float64
	_, offset := value.In(value.Location()).Zone()
	value = value.Add(time.Duration(offset) * time.Second)
	if excelTime, err = timeToExcelTime(value); err != nil {
		return
	}
	isNum = excelTime > 0
	if isNum {
		t, b = setCellDefault(strconv.FormatFloat(excelTime, 'f', -1, 64))
	} else {
		t, b = setCellDefault(value.Format(time.RFC3339Nano))
	}
	return
}

// setCellDuration prepares cell type and value by given Go time.Duration type
// time duration.
func setCellDuration(value time.Duration) (t string, v string) {
	v = strconv.FormatFloat(value.Seconds()/86400.0, 'f', -1, 32)
	return
}

// SetCellInt provides a function to set int type value of a cell by given
// worksheet name, cell coordinates and cell value.
func (f *File) SetCellInt(sheet, axis string, value int) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	cellData, col, row, err := f.prepareCell(ws, sheet, axis)
	if err != nil {
		return err
	}
	ws.Lock()
	defer ws.Unlock()
	cellData.S = f.prepareCellStyle(ws, col, row, cellData.S)
	cellData.T, cellData.V = setCellInt(value)
	return err
}

// setCellInt prepares cell type and string type cell value by a given
// integer.
func setCellInt(value int) (t string, v string) {
	v = strconv.Itoa(value)
	return
}

// SetCellBool provides a function to set bool type value of a cell by given
// worksheet name, cell name and cell value.
func (f *File) SetCellBool(sheet, axis string, value bool) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	cellData, col, row, err := f.prepareCell(ws, sheet, axis)
	if err != nil {
		return err
	}
	ws.Lock()
	defer ws.Unlock()
	cellData.S = f.prepareCellStyle(ws, col, row, cellData.S)
	cellData.T, cellData.V = setCellBool(value)
	return err
}

// setCellBool prepares cell type and string type cell value by a given
// boolean value.
func setCellBool(value bool) (t string, v string) {
	t = "b"
	if value {
		v = "1"
	} else {
		v = "0"
	}
	return
}

// SetCellFloat sets a floating point value into a cell. The prec parameter
// specifies how many places after the decimal will be shown while -1 is a
// special value that will use as many decimal places as necessary to
// represent the number. bitSize is 32 or 64 depending on if a float32 or
// float64 was originally used for the value. For Example:
//
//    var x float32 = 1.325
//    f.SetCellFloat("Sheet1", "A1", float64(x), 2, 32)
//
func (f *File) SetCellFloat(sheet, axis string, value float64, prec, bitSize int) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	cellData, col, row, err := f.prepareCell(ws, sheet, axis)
	if err != nil {
		return err
	}
	ws.Lock()
	defer ws.Unlock()
	cellData.S = f.prepareCellStyle(ws, col, row, cellData.S)
	cellData.T, cellData.V = setCellFloat(value, prec, bitSize)
	return err
}

// setCellFloat prepares cell type and string type cell value by a given
// float value.
func setCellFloat(value float64, prec, bitSize int) (t string, v string) {
	v = strconv.FormatFloat(value, 'f', prec, bitSize)
	return
}

// SetCellStr provides a function to set string type value of a cell. Total
// number of characters that a cell can contain 32767 characters.
func (f *File) SetCellStr(sheet, axis, value string) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	cellData, col, row, err := f.prepareCell(ws, sheet, axis)
	if err != nil {
		return err
	}
	ws.Lock()
	defer ws.Unlock()
	cellData.S = f.prepareCellStyle(ws, col, row, cellData.S)
	cellData.T, cellData.V, err = f.setCellString(value)
	return err
}

// setCellString provides a function to set string type to shared string
// table.
func (f *File) setCellString(value string) (t, v string, err error) {
	if len(value) > TotalCellChars {
		value = value[:TotalCellChars]
	}
	t = "s"
	var si int
	if si, err = f.setSharedString(value); err != nil {
		return
	}
	v = strconv.Itoa(si)
	return
}

// sharedStringsLoader load shared string table from system temporary file to
// memory, and reset shared string table for reader.
func (f *File) sharedStringsLoader() (err error) {
	f.Lock()
	defer f.Unlock()
	if path, ok := f.tempFiles.Load(defaultXMLPathSharedStrings); ok {
		f.Pkg.Store(defaultXMLPathSharedStrings, f.readBytes(defaultXMLPathSharedStrings))
		f.tempFiles.Delete(defaultXMLPathSharedStrings)
		if err = os.Remove(path.(string)); err != nil {
			return
		}
		f.SharedStrings = nil
	}
	if f.sharedStringTemp != nil {
		if err := f.sharedStringTemp.Close(); err != nil {
			return err
		}
		f.tempFiles.Delete(defaultTempFileSST)
		f.sharedStringItem, err = nil, os.Remove(f.sharedStringTemp.Name())
		f.sharedStringTemp = nil
	}
	return
}

// setSharedString provides a function to add string to the share string table.
func (f *File) setSharedString(val string) (int, error) {
	if err := f.sharedStringsLoader(); err != nil {
		return 0, err
	}
	sst := f.sharedStringsReader()
	f.Lock()
	defer f.Unlock()
	if i, ok := f.sharedStringsMap[val]; ok {
		return i, nil
	}
	sst.Count++
	sst.UniqueCount++
	t := xlsxT{Val: val}
	_, val, t.Space = setCellStr(val)
	sst.SI = append(sst.SI, xlsxSI{T: &t})
	f.sharedStringsMap[val] = sst.UniqueCount - 1
	return sst.UniqueCount - 1, nil
}

// setCellStr provides a function to set string type to cell.
func setCellStr(value string) (t string, v string, ns xml.Attr) {
	if len(value) > TotalCellChars {
		value = value[:TotalCellChars]
	}
	if len(value) > 0 {
		prefix, suffix := value[0], value[len(value)-1]
		for _, ascii := range []byte{9, 10, 13, 32} {
			if prefix == ascii || suffix == ascii {
				ns = xml.Attr{
					Name:  xml.Name{Space: NameSpaceXML, Local: "space"},
					Value: "preserve",
				}
				break
			}
		}
	}
	t, v = "str", bstrMarshal(value)
	return
}

// SetCellDefault provides a function to set string type value of a cell as
// default format without escaping the cell.
func (f *File) SetCellDefault(sheet, axis, value string) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	cellData, col, row, err := f.prepareCell(ws, sheet, axis)
	if err != nil {
		return err
	}
	ws.Lock()
	defer ws.Unlock()
	cellData.S = f.prepareCellStyle(ws, col, row, cellData.S)
	cellData.T, cellData.V = setCellDefault(value)
	return err
}

// setCellDefault prepares cell type and string type cell value by a given
// string.
func setCellDefault(value string) (t string, v string) {
	if ok, _ := isNumeric(value); !ok {
		t = "str"
	}
	v = value
	return
}

// GetCellFormula provides a function to get formula from cell by given
// worksheet name and axis in XLSX file.
func (f *File) GetCellFormula(sheet, axis string) (string, error) {
	return f.getCellStringFunc(sheet, axis, func(x *xlsxWorksheet, c *xlsxC) (string, bool, error) {
		if c.F == nil {
			return "", false, nil
		}
		if c.F.T == STCellFormulaTypeShared && c.F.Si != nil {
			return getSharedFormula(x, *c.F.Si, c.R), true, nil
		}
		return c.F.Content, true, nil
	})
}

// FormulaOpts can be passed to SetCellFormula to use other formula types.
type FormulaOpts struct {
	Type *string // Formula type
	Ref  *string // Shared formula ref
}

// SetCellFormula provides a function to set formula on the cell is taken
// according to the given worksheet name (case sensitive) and cell formula
// settings. The result of the formula cell can be calculated when the
// worksheet is opened by the Office Excel application or can be using
// the "CalcCellValue" function also can get the calculated cell value. If
// the Excel application doesn't calculate the formula automatically when the
// workbook has been opened, please call "UpdateLinkedValue" after setting
// the cell formula functions.
//
// Example 1, set normal formula "=SUM(A1,B1)" for the cell "A3" on "Sheet1":
//
//    err := f.SetCellFormula("Sheet1", "A3", "=SUM(A1,B1)")
//
// Example 2, set one-dimensional vertical constant array (row array) formula
// "1,2,3" for the cell "A3" on "Sheet1":
//
//    err := f.SetCellFormula("Sheet1", "A3", "={1,2,3}")
//
// Example 3, set one-dimensional horizontal constant array (column array)
// formula '"a","b","c"' for the cell "A3" on "Sheet1":
//
//    err := f.SetCellFormula("Sheet1", "A3", "={\"a\",\"b\",\"c\"}")
//
// Example 4, set two-dimensional constant array formula '{1,2,"a","b"}' for
// the cell "A3" on "Sheet1":
//
//    formulaType, ref := excelize.STCellFormulaTypeArray, "A3:A3"
//    err := f.SetCellFormula("Sheet1", "A3", "={1,2,\"a\",\"b\"}",
//        excelize.FormulaOpts{Ref: &ref, Type: &formulaType})
//
// Example 5, set range array formula "A1:A2" for the cell "A3" on "Sheet1":
//
//    formulaType, ref := excelize.STCellFormulaTypeArray, "A3:A3"
//    err := f.SetCellFormula("Sheet1", "A3", "=A1:A2",
//	      excelize.FormulaOpts{Ref: &ref, Type: &formulaType})
//
// Example 6, set shared formula "=A1+B1" for the cell "C1:C5"
// on "Sheet1", "C1" is the master cell:
//
//    formulaType, ref := excelize.STCellFormulaTypeShared, "C1:C5"
//    err := f.SetCellFormula("Sheet1", "C1", "=A1+B1",
//        excelize.FormulaOpts{Ref: &ref, Type: &formulaType})
//
// Example 7, set table formula "=SUM(Table1[[A]:[B]])" for the cell "C2"
// on "Sheet1":
//
//    package main
//
//    import (
//        "fmt"
//
//        "github.com/xuri/excelize/v2"
//    )
//
//    func main() {
//        f := excelize.NewFile()
//        for idx, row := range [][]interface{}{{"A", "B", "C"}, {1, 2}} {
//            if err := f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", idx+1), &row); err != nil {
//            	fmt.Println(err)
//            	return
//            }
//        }
//        if err := f.AddTable("Sheet1", "A1", "C2",
//            `{"table_name":"Table1","table_style":"TableStyleMedium2"}`); err != nil {
//            fmt.Println(err)
//            return
//        }
//        formulaType := excelize.STCellFormulaTypeDataTable
//        if err := f.SetCellFormula("Sheet1", "C2", "=SUM(Table1[[A]:[B]])",
//            excelize.FormulaOpts{Type: &formulaType}); err != nil {
//            fmt.Println(err)
//            return
//        }
//        if err := f.SaveAs("Book1.xlsx"); err != nil {
//            fmt.Println(err)
//        }
//    }
//
func (f *File) SetCellFormula(sheet, axis, formula string, opts ...FormulaOpts) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	cellData, _, _, err := f.prepareCell(ws, sheet, axis)
	if err != nil {
		return err
	}
	if formula == "" {
		cellData.F = nil
		f.deleteCalcChain(f.getSheetID(sheet), axis)
		return err
	}

	if cellData.F != nil {
		cellData.F.Content = formula
	} else {
		cellData.F = &xlsxF{Content: formula}
	}

	for _, o := range opts {
		if o.Type != nil {
			if *o.Type == STCellFormulaTypeDataTable {
				return err
			}
			cellData.F.T = *o.Type
			if cellData.F.T == STCellFormulaTypeShared {
				if err = ws.setSharedFormula(*o.Ref); err != nil {
					return err
				}
			}
		}
		if o.Ref != nil {
			cellData.F.Ref = *o.Ref
		}
	}

	return err
}

// setSharedFormula set shared formula for the cells.
func (ws *xlsxWorksheet) setSharedFormula(ref string) error {
	coordinates, err := areaRefToCoordinates(ref)
	if err != nil {
		return err
	}
	_ = sortCoordinates(coordinates)
	cnt := ws.countSharedFormula()
	for c := coordinates[0]; c <= coordinates[2]; c++ {
		for r := coordinates[1]; r <= coordinates[3]; r++ {
			prepareSheetXML(ws, c, r)
			cell := &ws.SheetData.Row[r-1].C[c-1]
			if cell.F == nil {
				cell.F = &xlsxF{}
			}
			cell.F.T = STCellFormulaTypeShared
			cell.F.Si = &cnt
		}
	}
	return err
}

// countSharedFormula count shared formula in the given worksheet.
func (ws *xlsxWorksheet) countSharedFormula() (count int) {
	for _, row := range ws.SheetData.Row {
		for _, cell := range row.C {
			if cell.F != nil && cell.F.Si != nil && *cell.F.Si+1 > count {
				count = *cell.F.Si + 1
			}
		}
	}
	return
}

// GetCellHyperLink provides a function to get cell hyperlink by given
// worksheet name and axis. Boolean type value link will be true if the cell
// has a hyperlink and the target is the address of the hyperlink. Otherwise,
// the value of link will be false and the value of the target will be a blank
// string. For example get hyperlink of Sheet1!H6:
//
//    link, target, err := f.GetCellHyperLink("Sheet1", "H6")
//
func (f *File) GetCellHyperLink(sheet, axis string) (bool, string, error) {
	// Check for correct cell name
	if _, _, err := SplitCellName(axis); err != nil {
		return false, "", err
	}
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return false, "", err
	}
	if axis, err = f.mergeCellsParser(ws, axis); err != nil {
		return false, "", err
	}
	if ws.Hyperlinks != nil {
		for _, link := range ws.Hyperlinks.Hyperlink {
			if link.Ref == axis {
				if link.RID != "" {
					return true, f.getSheetRelationshipsTargetByID(sheet, link.RID), err
				}
				return true, link.Location, err
			}
		}
	}
	return false, "", err
}

// HyperlinkOpts can be passed to SetCellHyperlink to set optional hyperlink
// attributes (e.g. display value)
type HyperlinkOpts struct {
	Display *string
	Tooltip *string
}

// SetCellHyperLink provides a function to set cell hyperlink by given
// worksheet name and link URL address. LinkType defines two types of
// hyperlink "External" for web site or "Location" for moving to one of cell
// in this workbook. Maximum limit hyperlinks in a worksheet is 65530. The
// below is example for external link.
//
//    if err := f.SetCellHyperLink("Sheet1", "A3",
//        "https://github.com/xuri/excelize", "External"); err != nil {
//        fmt.Println(err)
//    }
//    // Set underline and font color style for the cell.
//    style, err := f.NewStyle(&excelize.Style{
//        Font: &excelize.Font{Color: "#1265BE", Underline: "single"},
//    })
//    if err != nil {
//        fmt.Println(err)
//    }
//    err = f.SetCellStyle("Sheet1", "A3", "A3", style)
//
// A this is another example for "Location":
//
//    err := f.SetCellHyperLink("Sheet1", "A3", "Sheet1!A40", "Location")
//
func (f *File) SetCellHyperLink(sheet, axis, link, linkType string, opts ...HyperlinkOpts) error {
	// Check for correct cell name
	if _, _, err := SplitCellName(axis); err != nil {
		return err
	}

	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	if axis, err = f.mergeCellsParser(ws, axis); err != nil {
		return err
	}

	var linkData xlsxHyperlink

	if ws.Hyperlinks == nil {
		ws.Hyperlinks = new(xlsxHyperlinks)
	}

	if len(ws.Hyperlinks.Hyperlink) > TotalSheetHyperlinks {
		return ErrTotalSheetHyperlinks
	}

	switch linkType {
	case "External":
		linkData = xlsxHyperlink{
			Ref: axis,
		}
		sheetPath := f.sheetMap[trimSheetName(sheet)]
		sheetRels := "xl/worksheets/_rels/" + strings.TrimPrefix(sheetPath, "xl/worksheets/") + ".rels"
		rID := f.addRels(sheetRels, SourceRelationshipHyperLink, link, linkType)
		linkData.RID = "rId" + strconv.Itoa(rID)
		f.addSheetNameSpace(sheet, SourceRelationship)
	case "Location":
		linkData = xlsxHyperlink{
			Ref:      axis,
			Location: link,
		}
	default:
		return fmt.Errorf("invalid link type %q", linkType)
	}

	for _, o := range opts {
		if o.Display != nil {
			linkData.Display = *o.Display
		}
		if o.Tooltip != nil {
			linkData.Tooltip = *o.Tooltip
		}
	}

	ws.Hyperlinks.Hyperlink = append(ws.Hyperlinks.Hyperlink, linkData)
	return nil
}

// GetCellRichText provides a function to get rich text of cell by given
// worksheet.
func (f *File) GetCellRichText(sheet, cell string) (runs []RichTextRun, err error) {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return
	}
	cellData, _, _, err := f.prepareCell(ws, sheet, cell)
	if err != nil {
		return
	}
	siIdx, err := strconv.Atoi(cellData.V)
	if nil != err {
		return
	}
	sst := f.sharedStringsReader()
	if len(sst.SI) <= siIdx || siIdx < 0 {
		return
	}
	si := sst.SI[siIdx]
	for _, v := range si.R {
		run := RichTextRun{
			Text: v.T.Val,
		}
		if nil != v.RPr {
			font := Font{Underline: "none"}
			font.Bold = v.RPr.B != nil
			font.Italic = v.RPr.I != nil
			if v.RPr.U != nil {
				font.Underline = "single"
				if v.RPr.U.Val != nil {
					font.Underline = *v.RPr.U.Val
				}
			}
			if v.RPr.RFont != nil && v.RPr.RFont.Val != nil {
				font.Family = *v.RPr.RFont.Val
			}
			if v.RPr.Sz != nil && v.RPr.Sz.Val != nil {
				font.Size = *v.RPr.Sz.Val
			}
			font.Strike = v.RPr.Strike != nil
			if nil != v.RPr.Color {
				font.Color = strings.TrimPrefix(v.RPr.Color.RGB, "FF")
			}
			run.Font = &font
		}
		runs = append(runs, run)
	}
	return
}

// newRpr create run properties for the rich text by given font format.
func newRpr(fnt *Font) *xlsxRPr {
	rpr := xlsxRPr{}
	trueVal := ""
	if fnt.Bold {
		rpr.B = &trueVal
	}
	if fnt.Italic {
		rpr.I = &trueVal
	}
	if fnt.Strike {
		rpr.Strike = &trueVal
	}
	if fnt.Underline != "" {
		rpr.U = &attrValString{Val: &fnt.Underline}
	}
	if fnt.Family != "" {
		rpr.RFont = &attrValString{Val: &fnt.Family}
	}
	if fnt.Size > 0.0 {
		rpr.Sz = &attrValFloat{Val: &fnt.Size}
	}
	if fnt.Color != "" {
		rpr.Color = &xlsxColor{RGB: getPaletteColor(fnt.Color)}
	}
	return &rpr
}

// SetCellRichText provides a function to set cell with rich text by given
// worksheet. For example, set rich text on the A1 cell of the worksheet named
// Sheet1:
//
//    package main
//
//    import (
//        "fmt"
//
//        "github.com/xuri/excelize/v2"
//    )
//
//    func main() {
//        f := excelize.NewFile()
//        if err := f.SetRowHeight("Sheet1", 1, 35); err != nil {
//            fmt.Println(err)
//            return
//        }
//        if err := f.SetColWidth("Sheet1", "A", "A", 44); err != nil {
//            fmt.Println(err)
//            return
//        }
//        if err := f.SetCellRichText("Sheet1", "A1", []excelize.RichTextRun{
//            {
//                Text: "bold",
//                Font: &excelize.Font{
//                    Bold:   true,
//                    Color:  "2354e8",
//                    Family: "Times New Roman",
//                },
//            },
//            {
//                Text: " and ",
//                Font: &excelize.Font{
//                    Family: "Times New Roman",
//                },
//            },
//            {
//                Text: " italic",
//                Font: &excelize.Font{
//                    Bold:   true,
//                    Color:  "e83723",
//                    Italic: true,
//                    Family: "Times New Roman",
//                },
//            },
//            {
//                Text: "text with color and font-family,",
//                Font: &excelize.Font{
//                    Bold:   true,
//                    Color:  "2354e8",
//                    Family: "Times New Roman",
//                },
//            },
//            {
//                Text: "\r\nlarge text with ",
//                Font: &excelize.Font{
//                    Size:  14,
//                    Color: "ad23e8",
//                },
//            },
//            {
//                Text: "strike",
//                Font: &excelize.Font{
//                    Color:  "e89923",
//                    Strike: true,
//                },
//            },
//            {
//                Text: " and ",
//                Font: &excelize.Font{
//                    Size:  14,
//                    Color: "ad23e8",
//                },
//            },
//            {
//                Text: "underline.",
//                Font: &excelize.Font{
//                    Color:     "23e833",
//                    Underline: "single",
//                },
//            },
//        }); err != nil {
//            fmt.Println(err)
//            return
//        }
//        style, err := f.NewStyle(&excelize.Style{
//            Alignment: &excelize.Alignment{
//                WrapText: true,
//            },
//        })
//        if err != nil {
//            fmt.Println(err)
//            return
//        }
//        if err := f.SetCellStyle("Sheet1", "A1", "A1", style); err != nil {
//            fmt.Println(err)
//            return
//        }
//        if err := f.SaveAs("Book1.xlsx"); err != nil {
//            fmt.Println(err)
//        }
//    }
//
func (f *File) SetCellRichText(sheet, cell string, runs []RichTextRun) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	cellData, col, row, err := f.prepareCell(ws, sheet, cell)
	if err != nil {
		return err
	}
	if err := f.sharedStringsLoader(); err != nil {
		return err
	}
	cellData.S = f.prepareCellStyle(ws, col, row, cellData.S)
	si := xlsxSI{}
	sst := f.sharedStringsReader()
	textRuns := []xlsxR{}
	totalCellChars := 0
	for _, textRun := range runs {
		totalCellChars += len(textRun.Text)
		if totalCellChars > TotalCellChars {
			return ErrCellCharsLength
		}
		run := xlsxR{T: &xlsxT{}}
		_, run.T.Val, run.T.Space = setCellStr(textRun.Text)
		fnt := textRun.Font
		if fnt != nil {
			run.RPr = newRpr(fnt)
		}
		textRuns = append(textRuns, run)
	}
	si.R = textRuns
	for idx, strItem := range sst.SI {
		if reflect.DeepEqual(strItem, si) {
			cellData.T, cellData.V = "s", strconv.Itoa(idx)
			return err
		}
	}
	sst.SI = append(sst.SI, si)
	sst.Count++
	sst.UniqueCount++
	cellData.T, cellData.V = "s", strconv.Itoa(len(sst.SI)-1)
	return err
}

// SetSheetRow writes an array to row by given worksheet name, starting
// coordinate and a pointer to array type 'slice'. For example, writes an
// array to row 6 start with the cell B6 on Sheet1:
//
//     err := f.SetSheetRow("Sheet1", "B6", &[]interface{}{"1", nil, 2})
//
func (f *File) SetSheetRow(sheet, axis string, slice interface{}) error {
	col, row, err := CellNameToCoordinates(axis)
	if err != nil {
		return err
	}

	// Make sure 'slice' is a Ptr to Slice
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Slice {
		return ErrParameterInvalid
	}
	v = v.Elem()

	for i := 0; i < v.Len(); i++ {
		cell, err := CoordinatesToCellName(col+i, row)
		// Error should never happens here. But keep checking to early detect regresions
		// if it will be introduced in future.
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, v.Index(i).Interface()); err != nil {
			return err
		}
	}
	return err
}

// getCellInfo does common preparation for all SetCell* methods.
func (f *File) prepareCell(ws *xlsxWorksheet, sheet, cell string) (*xlsxC, int, int, error) {
	var err error
	cell, err = f.mergeCellsParser(ws, cell)
	if err != nil {
		return nil, 0, 0, err
	}
	col, row, err := CellNameToCoordinates(cell)
	if err != nil {
		return nil, 0, 0, err
	}

	prepareSheetXML(ws, col, row)
	ws.Lock()
	defer ws.Unlock()
	return &ws.SheetData.Row[row-1].C[col-1], col, row, err
}

// getCellStringFunc does common value extraction workflow for all GetCell*
// methods. Passed function implements specific part of required logic.
func (f *File) getCellStringFunc(sheet, axis string, fn func(x *xlsxWorksheet, c *xlsxC) (string, bool, error)) (string, error) {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return "", err
	}
	axis, err = f.mergeCellsParser(ws, axis)
	if err != nil {
		return "", err
	}
	_, row, err := CellNameToCoordinates(axis)
	if err != nil {
		return "", err
	}

	ws.Lock()
	defer ws.Unlock()

	lastRowNum := 0
	if l := len(ws.SheetData.Row); l > 0 {
		lastRowNum = ws.SheetData.Row[l-1].R
	}

	// keep in mind: row starts from 1
	if row > lastRowNum {
		return "", nil
	}

	for rowIdx := range ws.SheetData.Row {
		rowData := &ws.SheetData.Row[rowIdx]
		if rowData.R != row {
			continue
		}
		for colIdx := range rowData.C {
			colData := &rowData.C[colIdx]
			if axis != colData.R {
				continue
			}
			val, ok, err := fn(ws, colData)
			if err != nil {
				return "", err
			}
			if ok {
				return val, nil
			}
		}
	}
	return "", nil
}

// formattedValue provides a function to returns a value after formatted. If
// it is possible to apply a format to the cell value, it will do so, if not
// then an error will be returned, along with the raw value of the cell.
func (f *File) formattedValue(s int, v string, raw bool) string {
	if raw {
		return v
	}
	precise := v
	isNum, precision := isNumeric(v)
	if isNum {
		if precision > 15 {
			precise = roundPrecision(v, 15)
		}
		if precision <= 15 {
			precise = roundPrecision(v, -1)
		}
	}
	if s == 0 {
		return precise
	}
	styleSheet := f.stylesReader()
	if s >= len(styleSheet.CellXfs.Xf) {
		return precise
	}
	var numFmtID int
	if styleSheet.CellXfs.Xf[s].NumFmtID != nil {
		numFmtID = *styleSheet.CellXfs.Xf[s].NumFmtID
	}

	ok := builtInNumFmtFunc[numFmtID]
	if ok != nil {
		return ok(precise, builtInNumFmt[numFmtID])
	}
	if styleSheet == nil || styleSheet.NumFmts == nil {
		return precise
	}
	for _, xlsxFmt := range styleSheet.NumFmts.NumFmt {
		if xlsxFmt.NumFmtID == numFmtID {
			return format(precise, xlsxFmt.FormatCode)
		}
	}
	return precise
}

// prepareCellStyle provides a function to prepare style index of cell in
// worksheet by given column index and style index.
func (f *File) prepareCellStyle(ws *xlsxWorksheet, col, row, style int) int {
	if ws.Cols != nil && style == 0 {
		for _, c := range ws.Cols.Col {
			if c.Min <= col && col <= c.Max && c.Style != 0 {
				return c.Style
			}
		}
	}
	for rowIdx := range ws.SheetData.Row {
		if styleID := ws.SheetData.Row[rowIdx].S; style == 0 && styleID != 0 {
			return styleID
		}
	}
	return style
}

// mergeCellsParser provides a function to check merged cells in worksheet by
// given axis.
func (f *File) mergeCellsParser(ws *xlsxWorksheet, axis string) (string, error) {
	axis = strings.ToUpper(axis)
	if ws.MergeCells != nil {
		for i := 0; i < len(ws.MergeCells.Cells); i++ {
			if ws.MergeCells.Cells[i] == nil {
				ws.MergeCells.Cells = append(ws.MergeCells.Cells[:i], ws.MergeCells.Cells[i+1:]...)
				i--
				continue
			}
			ok, err := f.checkCellInArea(axis, ws.MergeCells.Cells[i].Ref)
			if err != nil {
				return axis, err
			}
			if ok {
				axis = strings.Split(ws.MergeCells.Cells[i].Ref, ":")[0]
			}
		}
	}
	return axis, nil
}

// checkCellInArea provides a function to determine if a given coordinate is
// within an area.
func (f *File) checkCellInArea(cell, area string) (bool, error) {
	col, row, err := CellNameToCoordinates(cell)
	if err != nil {
		return false, err
	}

	if rng := strings.Split(area, ":"); len(rng) != 2 {
		return false, err
	}
	coordinates, err := areaRefToCoordinates(area)
	if err != nil {
		return false, err
	}

	return cellInRef([]int{col, row}, coordinates), err
}

// cellInRef provides a function to determine if a given range is within an
// range.
func cellInRef(cell, ref []int) bool {
	return cell[0] >= ref[0] && cell[0] <= ref[2] && cell[1] >= ref[1] && cell[1] <= ref[3]
}

// isOverlap find if the given two rectangles overlap or not.
func isOverlap(rect1, rect2 []int) bool {
	return cellInRef([]int{rect1[0], rect1[1]}, rect2) ||
		cellInRef([]int{rect1[2], rect1[1]}, rect2) ||
		cellInRef([]int{rect1[0], rect1[3]}, rect2) ||
		cellInRef([]int{rect1[2], rect1[3]}, rect2) ||
		cellInRef([]int{rect2[0], rect2[1]}, rect1) ||
		cellInRef([]int{rect2[2], rect2[1]}, rect1) ||
		cellInRef([]int{rect2[0], rect2[3]}, rect1) ||
		cellInRef([]int{rect2[2], rect2[3]}, rect1)
}

// parseSharedFormula generate dynamic part of shared formula for target cell
// by given column and rows distance and origin shared formula.
func parseSharedFormula(dCol, dRow int, orig []byte) (res string, start int) {
	var (
		end           int
		stringLiteral bool
	)
	for end = 0; end < len(orig); end++ {
		c := orig[end]
		if c == '"' {
			stringLiteral = !stringLiteral
		}
		if stringLiteral {
			continue // Skip characters in quotes
		}
		if c >= 'A' && c <= 'Z' || c == '$' {
			res += string(orig[start:end])
			start = end
			end++
			foundNum := false
			for ; end < len(orig); end++ {
				idc := orig[end]
				if idc >= '0' && idc <= '9' || idc == '$' {
					foundNum = true
				} else if idc >= 'A' && idc <= 'Z' {
					if foundNum {
						break
					}
				} else {
					break
				}
			}
			if foundNum {
				cellID := string(orig[start:end])
				res += shiftCell(cellID, dCol, dRow)
				start = end
			}
		}
	}
	return
}

// getSharedFormula find a cell contains the same formula as another cell,
// the "shared" value can be used for the t attribute and the si attribute can
// be used to refer to the cell containing the formula. Two formulas are
// considered to be the same when their respective representations in
// R1C1-reference notation, are the same.
//
// Note that this function not validate ref tag to check the cell if or not in
// allow area, and always return origin shared formula.
func getSharedFormula(ws *xlsxWorksheet, si int, axis string) string {
	for _, r := range ws.SheetData.Row {
		for _, c := range r.C {
			if c.F != nil && c.F.Ref != "" && c.F.T == STCellFormulaTypeShared && c.F.Si != nil && *c.F.Si == si {
				col, row, _ := CellNameToCoordinates(axis)
				sharedCol, sharedRow, _ := CellNameToCoordinates(c.R)
				dCol := col - sharedCol
				dRow := row - sharedRow
				orig := []byte(c.F.Content)
				res, start := parseSharedFormula(dCol, dRow, orig)
				if start < len(orig) {
					res += string(orig[start:])
				}
				return res
			}
		}
	}
	return ""
}

// shiftCell returns the cell shifted according to dCol and dRow taking into
// consideration of absolute references with dollar sign ($)
func shiftCell(cellID string, dCol, dRow int) string {
	fCol, fRow, _ := CellNameToCoordinates(cellID)
	signCol, signRow := "", ""
	if strings.Index(cellID, "$") == 0 {
		signCol = "$"
	} else {
		// Shift column
		fCol += dCol
	}
	if strings.LastIndex(cellID, "$") > 0 {
		signRow = "$"
	} else {
		// Shift row
		fRow += dRow
	}
	colName, _ := ColumnNumberToName(fCol)
	return signCol + colName + signRow + strconv.Itoa(fRow)
}
