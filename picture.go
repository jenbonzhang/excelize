// Copyright 2016 - 2019 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to
// and read from XLSX files. Support reads and writes XLSX file generated by
// Microsoft Excel™ 2007 and later. Support save file without losing original
// charts of XLSX. This library needs Go version 1.8 or later.

package excelize

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"image"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// parseFormatPictureSet provides a function to parse the format settings of
// the picture with default value.
func parseFormatPictureSet(formatSet string) (*formatPicture, error) {
	format := formatPicture{
		FPrintsWithSheet: true,
		FLocksWithSheet:  false,
		NoChangeAspect:   false,
		OffsetX:          0,
		OffsetY:          0,
		XScale:           1.0,
		YScale:           1.0,
	}
	err := json.Unmarshal(parseFormatSet(formatSet), &format)
	return &format, err
}

// AddPicture provides the method to add picture in a sheet by given picture
// format set (such as offset, scale, aspect ratio setting and print settings)
// and file path. For example:
//
//    package main
//
//    import (
//        "fmt"
//        _ "image/gif"
//        _ "image/jpeg"
//        _ "image/png"
//
//        "github.com/360EntSecGroup-Skylar/excelize/v2"
//    )
//
//    func main() {
//        f := excelize.NewFile()
//        // Insert a picture.
//        err := f.AddPicture("Sheet1", "A2", "./image1.jpg", "")
//        if err != nil {
//            fmt.Println(err)
//        }
//        // Insert a picture scaling in the cell with location hyperlink.
//        err = f.AddPicture("Sheet1", "D2", "./image1.png", `{"x_scale": 0.5, "y_scale": 0.5, "hyperlink": "#Sheet2!D8", "hyperlink_type": "Location"}`)
//        if err != nil {
//            fmt.Println(err)
//        }
//        // Insert a picture offset in the cell with external hyperlink, printing and positioning support.
//        err = f.AddPicture("Sheet1", "H2", "./image3.gif", `{"x_offset": 15, "y_offset": 10, "hyperlink": "https://github.com/360EntSecGroup-Skylar/excelize", "hyperlink_type": "External", "print_obj": true, "lock_aspect_ratio": false, "locked": false, "positioning": "oneCell"}`)
//        if err != nil {
//            fmt.Println(err)
//        }
//        err = f.SaveAs("./Book1.xlsx")
//        if err != nil {
//            fmt.Println(err)
//        }
//    }
//
// LinkType defines two types of hyperlink "External" for web site or
// "Location" for moving to one of cell in this workbook. When the
// "hyperlink_type" is "Location", coordinates need to start with "#".
//
// Positioning defines two types of the position of a picture in an Excel
// spreadsheet, "oneCell" (Move but don't size with cells) or "absolute"
// (Don't move or size with cells). If you don't set this parameter, default
// positioning is move and size with cells.
func (f *File) AddPicture(sheet, cell, picture, format string) error {
	var err error
	// Check picture exists first.
	if _, err = os.Stat(picture); os.IsNotExist(err) {
		return err
	}
	ext, ok := supportImageTypes[path.Ext(picture)]
	if !ok {
		return errors.New("unsupported image extension")
	}
	file, _ := ioutil.ReadFile(picture)
	_, name := filepath.Split(picture)
	return f.AddPictureFromBytes(sheet, cell, format, name, ext, file)
}

// AddPictureFromBytes provides the method to add picture in a sheet by given
// picture format set (such as offset, scale, aspect ratio setting and print
// settings), file base name, extension name and file bytes. For example:
//
//    package main
//
//    import (
//        "fmt"
//        _ "image/jpeg"
//        "io/ioutil"
//
//        "github.com/360EntSecGroup-Skylar/excelize/v2"
//    )
//
//    func main() {
//        f := excelize.NewFile()
//
//        file, err := ioutil.ReadFile("./image1.jpg")
//        if err != nil {
//            fmt.Println(err)
//        }
//        err = f.AddPictureFromBytes("Sheet1", "A2", "", "Excel Logo", ".jpg", file)
//        if err != nil {
//            fmt.Println(err)
//        }
//        err = f.SaveAs("./Book1.xlsx")
//        if err != nil {
//            fmt.Println(err)
//        }
//    }
//
func (f *File) AddPictureFromBytes(sheet, cell, format, name, extension string, file []byte) error {
	var drawingHyperlinkRID int
	var hyperlinkType string
	ext, ok := supportImageTypes[extension]
	if !ok {
		return errors.New("unsupported image extension")
	}
	formatSet, err := parseFormatPictureSet(format)
	if err != nil {
		return err
	}
	img, _, err := image.DecodeConfig(bytes.NewReader(file))
	if err != nil {
		return err
	}
	// Read sheet data.
	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	// Add first picture for given sheet, create xl/drawings/ and xl/drawings/_rels/ folder.
	drawingID := f.countDrawings() + 1
	drawingXML := "xl/drawings/drawing" + strconv.Itoa(drawingID) + ".xml"
	drawingID, drawingXML = f.prepareDrawing(xlsx, drawingID, sheet, drawingXML)
	mediaStr := ".." + strings.TrimPrefix(f.addMedia(file, ext), "xl")
	drawingRID := f.addDrawingRelationships(drawingID, SourceRelationshipImage, mediaStr, hyperlinkType)
	// Add picture with hyperlink.
	if formatSet.Hyperlink != "" && formatSet.HyperlinkType != "" {
		if formatSet.HyperlinkType == "External" {
			hyperlinkType = formatSet.HyperlinkType
		}
		drawingHyperlinkRID = f.addDrawingRelationships(drawingID, SourceRelationshipHyperLink, formatSet.Hyperlink, hyperlinkType)
	}
	err = f.addDrawingPicture(sheet, drawingXML, cell, name, img.Width, img.Height, drawingRID, drawingHyperlinkRID, formatSet)
	if err != nil {
		return err
	}
	f.addContentTypePart(drawingID, "drawings")
	return err
}

// addSheetRelationships provides a function to add
// xl/worksheets/_rels/sheet%d.xml.rels by given worksheet name, relationship
// type and target.
func (f *File) addSheetRelationships(sheet, relType, target, targetMode string) int {
	name, ok := f.sheetMap[trimSheetName(sheet)]
	if !ok {
		name = strings.ToLower(sheet) + ".xml"
	}
	var rels = "xl/worksheets/_rels/" + strings.TrimPrefix(name, "xl/worksheets/") + ".rels"
	sheetRels := f.workSheetRelsReader(rels)
	if sheetRels == nil {
		sheetRels = &xlsxWorkbookRels{}
	}
	var rID = 1
	var ID bytes.Buffer
	ID.WriteString("rId")
	ID.WriteString(strconv.Itoa(rID))
	ID.Reset()
	rID = len(sheetRels.Relationships) + 1
	ID.WriteString("rId")
	ID.WriteString(strconv.Itoa(rID))
	sheetRels.Relationships = append(sheetRels.Relationships, xlsxWorkbookRelation{
		ID:         ID.String(),
		Type:       relType,
		Target:     target,
		TargetMode: targetMode,
	})
	f.WorkSheetRels[rels] = sheetRels
	return rID
}

// deleteSheetRelationships provides a function to delete relationships in
// xl/worksheets/_rels/sheet%d.xml.rels by given worksheet name and
// relationship index.
func (f *File) deleteSheetRelationships(sheet, rID string) {
	name, ok := f.sheetMap[trimSheetName(sheet)]
	if !ok {
		name = strings.ToLower(sheet) + ".xml"
	}
	var rels = "xl/worksheets/_rels/" + strings.TrimPrefix(name, "xl/worksheets/") + ".rels"
	sheetRels := f.workSheetRelsReader(rels)
	if sheetRels == nil {
		sheetRels = &xlsxWorkbookRels{}
	}
	for k, v := range sheetRels.Relationships {
		if v.ID == rID {
			sheetRels.Relationships = append(sheetRels.Relationships[:k], sheetRels.Relationships[k+1:]...)
		}
	}
	f.WorkSheetRels[rels] = sheetRels
}

// addSheetLegacyDrawing provides a function to add legacy drawing element to
// xl/worksheets/sheet%d.xml by given worksheet name and relationship index.
func (f *File) addSheetLegacyDrawing(sheet string, rID int) {
	xlsx, _ := f.workSheetReader(sheet)
	xlsx.LegacyDrawing = &xlsxLegacyDrawing{
		RID: "rId" + strconv.Itoa(rID),
	}
}

// addSheetDrawing provides a function to add drawing element to
// xl/worksheets/sheet%d.xml by given worksheet name and relationship index.
func (f *File) addSheetDrawing(sheet string, rID int) {
	xlsx, _ := f.workSheetReader(sheet)
	xlsx.Drawing = &xlsxDrawing{
		RID: "rId" + strconv.Itoa(rID),
	}
}

// addSheetPicture provides a function to add picture element to
// xl/worksheets/sheet%d.xml by given worksheet name and relationship index.
func (f *File) addSheetPicture(sheet string, rID int) {
	xlsx, _ := f.workSheetReader(sheet)
	xlsx.Picture = &xlsxPicture{
		RID: "rId" + strconv.Itoa(rID),
	}
}

// countDrawings provides a function to get drawing files count storage in the
// folder xl/drawings.
func (f *File) countDrawings() int {
	c1, c2 := 0, 0
	for k := range f.XLSX {
		if strings.Contains(k, "xl/drawings/drawing") {
			c1++
		}
	}
	for rel := range f.Drawings {
		if strings.Contains(rel, "xl/drawings/drawing") {
			c2++
		}
	}
	if c1 < c2 {
		return c2
	}
	return c1
}

// addDrawingPicture provides a function to add picture by given sheet,
// drawingXML, cell, file name, width, height relationship index and format
// sets.
func (f *File) addDrawingPicture(sheet, drawingXML, cell, file string, width, height, rID, hyperlinkRID int, formatSet *formatPicture) error {
	col, row, err := CellNameToCoordinates(cell)
	if err != nil {
		return err
	}
	width = int(float64(width) * formatSet.XScale)
	height = int(float64(height) * formatSet.YScale)
	col--
	row--
	colStart, rowStart, _, _, colEnd, rowEnd, x2, y2 :=
		f.positionObjectPixels(sheet, col, row, formatSet.OffsetX, formatSet.OffsetY, width, height)
	content, cNvPrID := f.drawingParser(drawingXML)
	twoCellAnchor := xdrCellAnchor{}
	twoCellAnchor.EditAs = formatSet.Positioning
	from := xlsxFrom{}
	from.Col = colStart
	from.ColOff = formatSet.OffsetX * EMU
	from.Row = rowStart
	from.RowOff = formatSet.OffsetY * EMU
	to := xlsxTo{}
	to.Col = colEnd
	to.ColOff = x2 * EMU
	to.Row = rowEnd
	to.RowOff = y2 * EMU
	twoCellAnchor.From = &from
	twoCellAnchor.To = &to
	pic := xlsxPic{}
	pic.NvPicPr.CNvPicPr.PicLocks.NoChangeAspect = formatSet.NoChangeAspect
	pic.NvPicPr.CNvPr.ID = f.countCharts() + f.countMedia() + 1
	pic.NvPicPr.CNvPr.Descr = file
	pic.NvPicPr.CNvPr.Name = "Picture " + strconv.Itoa(cNvPrID)
	if hyperlinkRID != 0 {
		pic.NvPicPr.CNvPr.HlinkClick = &xlsxHlinkClick{
			R:   SourceRelationship,
			RID: "rId" + strconv.Itoa(hyperlinkRID),
		}
	}
	pic.BlipFill.Blip.R = SourceRelationship
	pic.BlipFill.Blip.Embed = "rId" + strconv.Itoa(rID)
	pic.SpPr.PrstGeom.Prst = "rect"

	twoCellAnchor.Pic = &pic
	twoCellAnchor.ClientData = &xdrClientData{
		FLocksWithSheet:  formatSet.FLocksWithSheet,
		FPrintsWithSheet: formatSet.FPrintsWithSheet,
	}
	content.TwoCellAnchor = append(content.TwoCellAnchor, &twoCellAnchor)
	f.Drawings[drawingXML] = content
	return err
}

// addDrawingRelationships provides a function to add image part relationships
// in the file xl/drawings/_rels/drawing%d.xml.rels by given drawing index,
// relationship type and target.
func (f *File) addDrawingRelationships(index int, relType, target, targetMode string) int {
	var rels = "xl/drawings/_rels/drawing" + strconv.Itoa(index) + ".xml.rels"
	var rID = 1
	var ID bytes.Buffer
	ID.WriteString("rId")
	ID.WriteString(strconv.Itoa(rID))
	drawingRels := f.drawingRelsReader(rels)
	if drawingRels == nil {
		drawingRels = &xlsxWorkbookRels{}
	}
	ID.Reset()
	rID = len(drawingRels.Relationships) + 1
	ID.WriteString("rId")
	ID.WriteString(strconv.Itoa(rID))
	drawingRels.Relationships = append(drawingRels.Relationships, xlsxWorkbookRelation{
		ID:         ID.String(),
		Type:       relType,
		Target:     target,
		TargetMode: targetMode,
	})
	f.DrawingRels[rels] = drawingRels
	return rID
}

// countMedia provides a function to get media files count storage in the
// folder xl/media/image.
func (f *File) countMedia() int {
	count := 0
	for k := range f.XLSX {
		if strings.Contains(k, "xl/media/image") {
			count++
		}
	}
	return count
}

// addMedia provides a function to add a picture into folder xl/media/image by
// given file and extension name. Duplicate images are only actually stored once
// and drawings that use it will reference the same image.
func (f *File) addMedia(file []byte, ext string) string {
	count := f.countMedia()
	for name, existing := range f.XLSX {
		if !strings.HasPrefix(name, "xl/media/image") {
			continue
		}
		if bytes.Equal(file, existing) {
			return name
		}
	}
	media := "xl/media/image" + strconv.Itoa(count+1) + ext
	f.XLSX[media] = file
	return media
}

// setContentTypePartImageExtensions provides a function to set the content
// type for relationship parts and the Main Document part.
func (f *File) setContentTypePartImageExtensions() {
	var imageTypes = map[string]bool{"jpeg": false, "png": false, "gif": false}
	content := f.contentTypesReader()
	for _, v := range content.Defaults {
		_, ok := imageTypes[v.Extension]
		if ok {
			imageTypes[v.Extension] = true
		}
	}
	for k, v := range imageTypes {
		if !v {
			content.Defaults = append(content.Defaults, xlsxDefault{
				Extension:   k,
				ContentType: "image/" + k,
			})
		}
	}
}

// setContentTypePartVMLExtensions provides a function to set the content type
// for relationship parts and the Main Document part.
func (f *File) setContentTypePartVMLExtensions() {
	vml := false
	content := f.contentTypesReader()
	for _, v := range content.Defaults {
		if v.Extension == "vml" {
			vml = true
		}
	}
	if !vml {
		content.Defaults = append(content.Defaults, xlsxDefault{
			Extension:   "vml",
			ContentType: "application/vnd.openxmlformats-officedocument.vmlDrawing",
		})
	}
}

// addContentTypePart provides a function to add content type part
// relationships in the file [Content_Types].xml by given index.
func (f *File) addContentTypePart(index int, contentType string) {
	setContentType := map[string]func(){
		"comments": f.setContentTypePartVMLExtensions,
		"drawings": f.setContentTypePartImageExtensions,
	}
	partNames := map[string]string{
		"chart":    "/xl/charts/chart" + strconv.Itoa(index) + ".xml",
		"comments": "/xl/comments" + strconv.Itoa(index) + ".xml",
		"drawings": "/xl/drawings/drawing" + strconv.Itoa(index) + ".xml",
		"table":    "/xl/tables/table" + strconv.Itoa(index) + ".xml",
	}
	contentTypes := map[string]string{
		"chart":    "application/vnd.openxmlformats-officedocument.drawingml.chart+xml",
		"comments": "application/vnd.openxmlformats-officedocument.spreadsheetml.comments+xml",
		"drawings": "application/vnd.openxmlformats-officedocument.drawing+xml",
		"table":    "application/vnd.openxmlformats-officedocument.spreadsheetml.table+xml",
	}
	s, ok := setContentType[contentType]
	if ok {
		s()
	}
	content := f.contentTypesReader()
	for _, v := range content.Overrides {
		if v.PartName == partNames[contentType] {
			return
		}
	}
	content.Overrides = append(content.Overrides, xlsxOverride{
		PartName:    partNames[contentType],
		ContentType: contentTypes[contentType],
	})
}

// getSheetRelationshipsTargetByID provides a function to get Target attribute
// value in xl/worksheets/_rels/sheet%d.xml.rels by given worksheet name and
// relationship index.
func (f *File) getSheetRelationshipsTargetByID(sheet, rID string) string {
	name, ok := f.sheetMap[trimSheetName(sheet)]
	if !ok {
		name = strings.ToLower(sheet) + ".xml"
	}
	var rels = "xl/worksheets/_rels/" + strings.TrimPrefix(name, "xl/worksheets/") + ".rels"
	sheetRels := f.workSheetRelsReader(rels)
	if sheetRels == nil {
		sheetRels = &xlsxWorkbookRels{}
	}
	for _, v := range sheetRels.Relationships {
		if v.ID == rID {
			return v.Target
		}
	}
	return ""
}

// GetPicture provides a function to get picture base name and raw content
// embed in XLSX by given worksheet and cell name. This function returns the
// file name in XLSX and file contents as []byte data types. For example:
//
//    f, err := excelize.OpenFile("./Book1.xlsx")
//    if err != nil {
//        fmt.Println(err)
//        return
//    }
//    file, raw, err := f.GetPicture("Sheet1", "A2")
//    if err != nil {
//        fmt.Println(err)
//        return
//    }
//    err = ioutil.WriteFile(file, raw, 0644)
//    if err != nil {
//        fmt.Println(err)
//    }
//
func (f *File) GetPicture(sheet, cell string) (string, []byte, error) {
	col, row, err := CellNameToCoordinates(cell)
	if err != nil {
		return "", nil, err
	}
	col--
	row--
	xlsx, err := f.workSheetReader(sheet)
	if err != nil {
		return "", nil, err
	}
	if xlsx.Drawing == nil {
		return "", nil, err
	}
	target := f.getSheetRelationshipsTargetByID(sheet, xlsx.Drawing.RID)
	drawingXML := strings.Replace(target, "..", "xl", -1)
	_, ok := f.XLSX[drawingXML]
	if !ok {
		return "", nil, err
	}
	drawingRelationships := strings.Replace(
		strings.Replace(target, "../drawings", "xl/drawings/_rels", -1), ".xml", ".xml.rels", -1)

	return f.getPicture(row, col, drawingXML, drawingRelationships)
}

// getPicture provides a function to get picture base name and raw content
// embed in XLSX by given coordinates and drawing relationships.
func (f *File) getPicture(row, col int, drawingXML, drawingRelationships string) (string, []byte, error) {
	wsDr, _ := f.drawingParser(drawingXML)
	for _, anchor := range wsDr.TwoCellAnchor {
		if anchor.From != nil && anchor.Pic != nil {
			if anchor.From.Col == col && anchor.From.Row == row {
				xlsxWorkbookRelation := f.getDrawingRelationships(drawingRelationships,
					anchor.Pic.BlipFill.Blip.Embed)
				_, ok := supportImageTypes[filepath.Ext(xlsxWorkbookRelation.Target)]
				if ok {
					return filepath.Base(xlsxWorkbookRelation.Target),
						[]byte(f.XLSX[strings.Replace(xlsxWorkbookRelation.Target,
							"..", "xl", -1)]), nil
				}
			}
		}
	}

	decodeWsDr := decodeWsDr{}
	_ = xml.Unmarshal(namespaceStrictToTransitional(f.readXML(drawingXML)), &decodeWsDr)
	for _, anchor := range decodeWsDr.TwoCellAnchor {
		decodeTwoCellAnchor := decodeTwoCellAnchor{}
		_ = xml.Unmarshal([]byte("<decodeTwoCellAnchor>"+anchor.Content+"</decodeTwoCellAnchor>"), &decodeTwoCellAnchor)
		if decodeTwoCellAnchor.From != nil && decodeTwoCellAnchor.Pic != nil {
			if decodeTwoCellAnchor.From.Col == col && decodeTwoCellAnchor.From.Row == row {
				xlsxWorkbookRelation := f.getDrawingRelationships(drawingRelationships, decodeTwoCellAnchor.Pic.BlipFill.Blip.Embed)
				_, ok := supportImageTypes[filepath.Ext(xlsxWorkbookRelation.Target)]
				if ok {
					return filepath.Base(xlsxWorkbookRelation.Target), []byte(f.XLSX[strings.Replace(xlsxWorkbookRelation.Target, "..", "xl", -1)]), nil
				}
			}
		}
	}
	return "", nil, nil
}

// getDrawingRelationships provides a function to get drawing relationships
// from xl/drawings/_rels/drawing%s.xml.rels by given file name and
// relationship ID.
func (f *File) getDrawingRelationships(rels, rID string) *xlsxWorkbookRelation {
	if drawingRels := f.drawingRelsReader(rels); drawingRels != nil {
		for _, v := range drawingRels.Relationships {
			if v.ID == rID {
				return &v
			}
		}
	}
	return nil
}

// drawingRelsReader provides a function to get the pointer to the structure
// after deserialization of xl/drawings/_rels/drawing%d.xml.rels.
func (f *File) drawingRelsReader(rel string) *xlsxWorkbookRels {
	if f.DrawingRels[rel] == nil {
		_, ok := f.XLSX[rel]
		if ok {
			d := xlsxWorkbookRels{}
			_ = xml.Unmarshal(namespaceStrictToTransitional(f.readXML(rel)), &d)
			f.DrawingRels[rel] = &d
		}
	}
	return f.DrawingRels[rel]
}

// drawingRelsWriter provides a function to save
// xl/drawings/_rels/drawing%d.xml.rels after serialize structure.
func (f *File) drawingRelsWriter() {
	for path, d := range f.DrawingRels {
		if d != nil {
			v, _ := xml.Marshal(d)
			f.saveFileList(path, v)
		}
	}
}

// drawingsWriter provides a function to save xl/drawings/drawing%d.xml after
// serialize structure.
func (f *File) drawingsWriter() {
	for path, d := range f.Drawings {
		if d != nil {
			v, _ := xml.Marshal(d)
			f.saveFileList(path, v)
		}
	}
}
