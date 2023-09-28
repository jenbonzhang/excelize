// Copyright 2016 - 2023 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to and
// read from XLAM / XLSM / XLSX / XLTM / XLTX files. Supports reading and
// writing spreadsheet documents generated by Microsoft Excel™ 2007 and later.
// Supports complex components by high compatibility, and provided streaming
// API for generating or reading data from a worksheet with huge amounts of
// data. This library needs Go version 1.16 or later.

package excelize

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

// FormControlType is the type of supported form controls.
type FormControlType byte

// This section defines the currently supported form control types enumeration.
const (
	FormControlNote FormControlType = iota
	FormControlButton
	FormControlOptionButton
	FormControlSpinButton
	FormControlCheckBox
	FormControlGroupBox
	FormControlLabel
	FormControlScrollBar
)

// GetComments retrieves all comments in a worksheet by given worksheet name.
func (f *File) GetComments(sheet string) ([]Comment, error) {
	var comments []Comment
	sheetXMLPath, ok := f.getSheetXMLPath(sheet)
	if !ok {
		return comments, ErrSheetNotExist{sheet}
	}
	commentsXML := f.getSheetComments(filepath.Base(sheetXMLPath))
	if !strings.HasPrefix(commentsXML, "/") {
		commentsXML = "xl" + strings.TrimPrefix(commentsXML, "..")
	}
	commentsXML = strings.TrimPrefix(commentsXML, "/")
	cmts, err := f.commentsReader(commentsXML)
	if err != nil {
		return comments, err
	}
	if cmts != nil {
		for _, cmt := range cmts.CommentList.Comment {
			comment := Comment{}
			if cmt.AuthorID < len(cmts.Authors.Author) {
				comment.Author = cmts.Authors.Author[cmt.AuthorID]
			}
			comment.Cell = cmt.Ref
			comment.AuthorID = cmt.AuthorID
			if cmt.Text.T != nil {
				comment.Text += *cmt.Text.T
			}
			for _, text := range cmt.Text.R {
				if text.T != nil {
					run := RichTextRun{Text: text.T.Val}
					if text.RPr != nil {
						run.Font = newFont(text.RPr)
					}
					comment.Paragraph = append(comment.Paragraph, run)
				}
			}
			comments = append(comments, comment)
		}
	}
	return comments, nil
}

// getSheetComments provides the method to get the target comment reference by
// given worksheet file path.
func (f *File) getSheetComments(sheetFile string) string {
	rels, _ := f.relsReader("xl/worksheets/_rels/" + sheetFile + ".rels")
	if sheetRels := rels; sheetRels != nil {
		sheetRels.mu.Lock()
		defer sheetRels.mu.Unlock()
		for _, v := range sheetRels.Relationships {
			if v.Type == SourceRelationshipComments {
				return v.Target
			}
		}
	}
	return ""
}

// AddComment provides the method to add comment in a sheet by given worksheet
// name, cell reference and format set (such as author and text). Note that the
// max author length is 255 and the max text length is 32512. For example, add
// a comment in Sheet1!$A$30:
//
//	err := f.AddComment("Sheet1", excelize.Comment{
//	    Cell:   "A12",
//	    Author: "Excelize",
//	    Paragraph: []excelize.RichTextRun{
//	        {Text: "Excelize: ", Font: &excelize.Font{Bold: true}},
//	        {Text: "This is a comment."},
//	    },
//	})
func (f *File) AddComment(sheet string, opts Comment) error {
	return f.addVMLObject(vmlOptions{
		sheet: sheet, Comment: opts,
		FormControl: FormControl{Cell: opts.Cell, Type: FormControlNote},
	})
}

// DeleteComment provides the method to delete comment in a worksheet by given
// worksheet name and cell reference. For example, delete the comment in
// Sheet1!$A$30:
//
//	err := f.DeleteComment("Sheet1", "A30")
func (f *File) DeleteComment(sheet, cell string) error {
	if err := checkSheetName(sheet); err != nil {
		return err
	}
	sheetXMLPath, ok := f.getSheetXMLPath(sheet)
	if !ok {
		return ErrSheetNotExist{sheet}
	}
	commentsXML := f.getSheetComments(filepath.Base(sheetXMLPath))
	if !strings.HasPrefix(commentsXML, "/") {
		commentsXML = "xl" + strings.TrimPrefix(commentsXML, "..")
	}
	commentsXML = strings.TrimPrefix(commentsXML, "/")
	cmts, err := f.commentsReader(commentsXML)
	if err != nil {
		return err
	}
	if cmts != nil {
		for i := 0; i < len(cmts.CommentList.Comment); i++ {
			cmt := cmts.CommentList.Comment[i]
			if cmt.Ref != cell {
				continue
			}
			if len(cmts.CommentList.Comment) > 1 {
				cmts.CommentList.Comment = append(
					cmts.CommentList.Comment[:i],
					cmts.CommentList.Comment[i+1:]...,
				)
				i--
				continue
			}
			cmts.CommentList.Comment = nil
		}
		f.Comments[commentsXML] = cmts
	}
	return err
}

// addComment provides a function to create chart as xl/comments%d.xml by
// given cell and format sets.
func (f *File) addComment(commentsXML string, opts vmlOptions) error {
	if opts.Author == "" {
		opts.Author = "Author"
	}
	if len(opts.Author) > MaxFieldLength {
		opts.Author = opts.Author[:MaxFieldLength]
	}
	cmts, err := f.commentsReader(commentsXML)
	if err != nil {
		return err
	}
	var authorID int
	if cmts == nil {
		cmts = &xlsxComments{Authors: xlsxAuthor{Author: []string{opts.Author}}}
	}
	if inStrSlice(cmts.Authors.Author, opts.Author, true) == -1 {
		cmts.Authors.Author = append(cmts.Authors.Author, opts.Author)
		authorID = len(cmts.Authors.Author) - 1
	}
	defaultFont, err := f.GetDefaultFont()
	if err != nil {
		return err
	}
	chars, cmt := 0, xlsxComment{
		Ref:      opts.Comment.Cell,
		AuthorID: authorID,
		Text:     xlsxText{R: []xlsxR{}},
	}
	if opts.Comment.Text != "" {
		if len(opts.Comment.Text) > TotalCellChars {
			opts.Comment.Text = opts.Comment.Text[:TotalCellChars]
		}
		cmt.Text.T = stringPtr(opts.Comment.Text)
		chars += len(opts.Comment.Text)
	}
	for _, run := range opts.Comment.Paragraph {
		if chars == TotalCellChars {
			break
		}
		if chars+len(run.Text) > TotalCellChars {
			run.Text = run.Text[:TotalCellChars-chars]
		}
		chars += len(run.Text)
		r := xlsxR{
			RPr: &xlsxRPr{
				Sz: &attrValFloat{Val: float64Ptr(9)},
				Color: &xlsxColor{
					Indexed: 81,
				},
				RFont:  &attrValString{Val: stringPtr(defaultFont)},
				Family: &attrValInt{Val: intPtr(2)},
			},
			T: &xlsxT{Val: run.Text, Space: xml.Attr{
				Name:  xml.Name{Space: NameSpaceXML, Local: "space"},
				Value: "preserve",
			}},
		}
		if run.Font != nil {
			r.RPr = newRpr(run.Font)
		}
		cmt.Text.R = append(cmt.Text.R, r)
	}
	cmts.CommentList.Comment = append(cmts.CommentList.Comment, cmt)
	f.Comments[commentsXML] = cmts
	return err
}

// countComments provides a function to get comments files count storage in
// the folder xl.
func (f *File) countComments() int {
	comments := map[string]struct{}{}
	f.Pkg.Range(func(k, v interface{}) bool {
		if strings.Contains(k.(string), "xl/comments") {
			comments[k.(string)] = struct{}{}
		}
		return true
	})
	for rel := range f.Comments {
		if strings.Contains(rel, "xl/comments") {
			comments[rel] = struct{}{}
		}
	}
	return len(comments)
}

// commentsReader provides a function to get the pointer to the structure
// after deserialization of xl/comments%d.xml.
func (f *File) commentsReader(path string) (*xlsxComments, error) {
	if f.Comments[path] == nil {
		content, ok := f.Pkg.Load(path)
		if ok && content != nil {
			f.Comments[path] = new(xlsxComments)
			if err := f.xmlNewDecoder(bytes.NewReader(namespaceStrictToTransitional(content.([]byte)))).
				Decode(f.Comments[path]); err != nil && err != io.EOF {
				return nil, err
			}
		}
	}
	return f.Comments[path], nil
}

// commentsWriter provides a function to save xl/comments%d.xml after
// serialize structure.
func (f *File) commentsWriter() {
	for path, c := range f.Comments {
		if c != nil {
			v, _ := xml.Marshal(c)
			f.saveFileList(path, v)
		}
	}
}

// AddFormControl provides the method to add form control button in a worksheet
// by given worksheet name and form control options. Supported form control
// type: button, check box, group box, label, option button, scroll bar and
// spinner. If set macro for the form control, the workbook extension should be
// XLSM or XLTM. Scroll value must be between 0 and 30000.
//
// Example 1, add button form control with macro, rich-text, custom button size,
// print property on Sheet1!A2, and let the button do not move or size with
// cells:
//
//	enable := true
//	err := f.AddFormControl("Sheet1", excelize.FormControl{
//	    Cell:   "A2",
//	    Type:   excelize.FormControlButton,
//	    Macro:  "Button1_Click",
//	    Width:  140,
//	    Height: 60,
//	    Text:   "Button 1\r\n",
//	    Paragraph: []excelize.RichTextRun{
//	        {
//	            Font: &excelize.Font{
//	                Bold:      true,
//	                Italic:    true,
//	                Underline: "single",
//	                Family:    "Times New Roman",
//	                Size:      14,
//	                Color:     "777777",
//	            },
//	            Text: "C1=A1+B1",
//	        },
//	    },
//	    Format: excelize.GraphicOptions{
//	        PrintObject: &enable,
//	        Positioning: "absolute",
//	    },
//	})
//
// Example 2, add option button form control with checked status and text on
// Sheet1!A1:
//
//	err := f.AddFormControl("Sheet1", excelize.FormControl{
//	    Cell:    "A1",
//	    Type:    excelize.FormControlOptionButton,
//	    Text:    "Option Button 1",
//	    Checked: true,
//	})
//
// Example 3, add spin button form control on Sheet1!B1 to increase or decrease
// the value of Sheet1!A1:
//
//	err := f.AddFormControl("Sheet1", excelize.FormControl{
//	    Cell:       "B1",
//	    Type:       excelize.FormControlSpinButton,
//	    Width:      15,
//	    Height:     40,
//	    CurrentVal: 7,
//	    MinVal:     5,
//	    MaxVal:     10,
//	    IncChange:  1,
//	    CellLink:   "A1",
//	})
//
// Example 4, add horizontally scroll bar form control on Sheet1!A2 to change
// the value of Sheet1!A1 by click the scroll arrows or drag the scroll box:
//
//	err := f.AddFormControl("Sheet1", excelize.FormControl{
//	    Cell:         "A2",
//	    Type:         excelize.FormControlScrollBar,
//	    Width:        140,
//	    Height:       20,
//	    CurrentVal:   50,
//	    MinVal:       10,
//	    MaxVal:       100,
//	    IncChange:    1,
//	    PageChange:   1,
//	    CellLink:     "A1",
//	    Horizontally: true,
//	})
func (f *File) AddFormControl(sheet string, opts FormControl) error {
	return f.addVMLObject(vmlOptions{
		formCtrl: true, sheet: sheet, FormControl: opts,
	})
}

// DeleteFormControl provides the method to delete form control in a worksheet
// by given worksheet name and cell reference. For example, delete the form
// control in Sheet1!$A$1:
//
//	err := f.DeleteFormControl("Sheet1", "A1")
func (f *File) DeleteFormControl(sheet, cell string) error {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return err
	}
	col, row, err := CellNameToCoordinates(cell)
	if err != nil {
		return err
	}
	if ws.LegacyDrawing == nil {
		return err
	}
	sheetRelationshipsDrawingVML := f.getSheetRelationshipsTargetByID(sheet, ws.LegacyDrawing.RID)
	vmlID, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(sheetRelationshipsDrawingVML, "../drawings/vmlDrawing"), ".vml"))
	drawingVML := strings.ReplaceAll(sheetRelationshipsDrawingVML, "..", "xl")
	vml := f.VMLDrawing[drawingVML]
	if vml == nil {
		vml = &vmlDrawing{
			XMLNSv:  "urn:schemas-microsoft-com:vml",
			XMLNSo:  "urn:schemas-microsoft-com:office:office",
			XMLNSx:  "urn:schemas-microsoft-com:office:excel",
			XMLNSmv: "http://macVmlSchemaUri",
			ShapeLayout: &xlsxShapeLayout{
				Ext: "edit", IDmap: &xlsxIDmap{Ext: "edit", Data: vmlID},
			},
			ShapeType: &xlsxShapeType{
				Stroke: &xlsxStroke{JoinStyle: "miter"},
				VPath:  &vPath{GradientShapeOK: "t", ConnectType: "rect"},
			},
		}
		// Load exist VML shapes from xl/drawings/vmlDrawing%d.vml
		d, err := f.decodeVMLDrawingReader(drawingVML)
		if err != nil {
			return err
		}
		if d != nil {
			vml.ShapeType.ID = d.ShapeType.ID
			vml.ShapeType.CoordSize = d.ShapeType.CoordSize
			vml.ShapeType.Spt = d.ShapeType.Spt
			vml.ShapeType.Path = d.ShapeType.Path
			for _, v := range d.Shape {
				s := xlsxShape{
					ID:          v.ID,
					Type:        v.Type,
					Style:       v.Style,
					Button:      v.Button,
					Filled:      v.Filled,
					FillColor:   v.FillColor,
					InsetMode:   v.InsetMode,
					Stroked:     v.Stroked,
					StrokeColor: v.StrokeColor,
					Val:         v.Val,
				}
				vml.Shape = append(vml.Shape, s)
			}
		}
	}
	for i, sp := range vml.Shape {
		var shapeVal decodeShapeVal
		if err = xml.Unmarshal([]byte(fmt.Sprintf("<shape>%s</shape>", sp.Val)), &shapeVal); err == nil &&
			shapeVal.ClientData.ObjectType != "Note" && shapeVal.ClientData.Anchor != "" {
			leftCol, topRow, err := extractAnchorCell(shapeVal.ClientData.Anchor)
			if err != nil {
				return err
			}
			if leftCol == col-1 && topRow == row-1 {
				vml.Shape = append(vml.Shape[:i], vml.Shape[i+1:]...)
				break
			}
		}
	}
	f.VMLDrawing[drawingVML] = vml
	return err
}

// countVMLDrawing provides a function to get VML drawing files count storage
// in the folder xl/drawings.
func (f *File) countVMLDrawing() int {
	drawings := map[string]struct{}{}
	f.Pkg.Range(func(k, v interface{}) bool {
		if strings.Contains(k.(string), "xl/drawings/vmlDrawing") {
			drawings[k.(string)] = struct{}{}
		}
		return true
	})
	for rel := range f.VMLDrawing {
		if strings.Contains(rel, "xl/drawings/vmlDrawing") {
			drawings[rel] = struct{}{}
		}
	}
	return len(drawings)
}

// decodeVMLDrawingReader provides a function to get the pointer to the
// structure after deserialization of xl/drawings/vmlDrawing%d.xml.
func (f *File) decodeVMLDrawingReader(path string) (*decodeVmlDrawing, error) {
	if f.DecodeVMLDrawing[path] == nil {
		c, ok := f.Pkg.Load(path)
		if ok && c != nil {
			f.DecodeVMLDrawing[path] = new(decodeVmlDrawing)
			if err := f.xmlNewDecoder(bytes.NewReader(bytesReplace(namespaceStrictToTransitional(c.([]byte)), []byte("<br>\r\n"), []byte("<br></br>\r\n"), -1))).
				Decode(f.DecodeVMLDrawing[path]); err != nil && err != io.EOF {
				return nil, err
			}
		}
	}
	return f.DecodeVMLDrawing[path], nil
}

// vmlDrawingWriter provides a function to save xl/drawings/vmlDrawing%d.xml
// after serialize structure.
func (f *File) vmlDrawingWriter() {
	for path, vml := range f.VMLDrawing {
		if vml != nil {
			v, _ := xml.Marshal(vml)
			f.Pkg.Store(path, v)
		}
	}
}

// addVMLObject provides a function to create VML drawing parts and
// relationships for comments and form controls.
func (f *File) addVMLObject(opts vmlOptions) error {
	// Read sheet data
	ws, err := f.workSheetReader(opts.sheet)
	if err != nil {
		return err
	}
	vmlID := f.countComments() + 1
	if opts.formCtrl {
		if opts.Type > FormControlScrollBar {
			return ErrParameterInvalid
		}
		vmlID = f.countVMLDrawing() + 1
	}
	drawingVML := "xl/drawings/vmlDrawing" + strconv.Itoa(vmlID) + ".vml"
	sheetRelationshipsDrawingVML := "../drawings/vmlDrawing" + strconv.Itoa(vmlID) + ".vml"
	sheetXMLPath, _ := f.getSheetXMLPath(opts.sheet)
	sheetRels := "xl/worksheets/_rels/" + strings.TrimPrefix(sheetXMLPath, "xl/worksheets/") + ".rels"
	if ws.LegacyDrawing != nil {
		// The worksheet already has a VML relationships, use the relationships drawing ../drawings/vmlDrawing%d.vml.
		sheetRelationshipsDrawingVML = f.getSheetRelationshipsTargetByID(opts.sheet, ws.LegacyDrawing.RID)
		vmlID, _ = strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(sheetRelationshipsDrawingVML, "../drawings/vmlDrawing"), ".vml"))
		drawingVML = strings.ReplaceAll(sheetRelationshipsDrawingVML, "..", "xl")
	} else {
		// Add first VML drawing for given sheet.
		rID := f.addRels(sheetRels, SourceRelationshipDrawingVML, sheetRelationshipsDrawingVML, "")
		f.addSheetNameSpace(opts.sheet, SourceRelationship)
		f.addSheetLegacyDrawing(opts.sheet, rID)
	}
	if err = f.addDrawingVML(vmlID, drawingVML, prepareFormCtrlOptions(&opts)); err != nil {
		return err
	}
	if !opts.formCtrl {
		commentsXML := "xl/comments" + strconv.Itoa(vmlID) + ".xml"
		if err = f.addComment(commentsXML, opts); err != nil {
			return err
		}
		if sheetXMLPath, ok := f.getSheetXMLPath(opts.sheet); ok && f.getSheetComments(filepath.Base(sheetXMLPath)) == "" {
			sheetRelationshipsComments := "../comments" + strconv.Itoa(vmlID) + ".xml"
			f.addRels(sheetRels, SourceRelationshipComments, sheetRelationshipsComments, "")
		}
	}
	return f.addContentTypePart(vmlID, "comments")
}

// prepareFormCtrlOptions provides a function to parse the format settings of
// the form control with default value.
func prepareFormCtrlOptions(opts *vmlOptions) *vmlOptions {
	for _, runs := range opts.FormControl.Paragraph {
		for _, subStr := range strings.Split(runs.Text, "\n") {
			opts.rows++
			if chars := len(subStr); chars > opts.cols {
				opts.cols = chars
			}
		}
	}
	if len(opts.FormControl.Paragraph) == 0 {
		opts.rows, opts.cols = 1, len(opts.FormControl.Text)
	}
	if opts.Format.ScaleX == 0 {
		opts.Format.ScaleX = 1
	}
	if opts.Format.ScaleY == 0 {
		opts.Format.ScaleY = 1
	}
	if opts.cols == 0 {
		opts.cols = 8
	}
	if opts.Width == 0 {
		opts.Width = uint(opts.cols * 9)
	}
	if opts.Height == 0 {
		opts.Height = uint(opts.rows * 25)
	}
	return opts
}

// formCtrlText returns font element in the VML for control form text.
func formCtrlText(opts *vmlOptions) []vmlFont {
	var font []vmlFont
	if opts.FormControl.Text != "" {
		font = append(font, vmlFont{Content: opts.FormControl.Text})
	}
	for _, run := range opts.FormControl.Paragraph {
		fnt := vmlFont{
			Content: run.Text + "<br></br>\r\n",
		}
		if run.Font != nil {
			fnt.Face = run.Font.Family
			fnt.Color = run.Font.Color
			if !strings.HasPrefix(run.Font.Color, "#") {
				fnt.Color = "#" + fnt.Color
			}
			if run.Font.Size != 0 {
				fnt.Size = uint(run.Font.Size * 20)
			}
			if run.Font.Underline == "single" {
				fnt.Content = "<u>" + fnt.Content + "</u>"
			}
			if run.Font.Underline == "double" {
				fnt.Content = "<u class=\"font1\">" + fnt.Content + "</u>"
			}
			if run.Font.Italic {
				fnt.Content = "<i>" + fnt.Content + "</i>"
			}
			if run.Font.Bold {
				fnt.Content = "<b>" + fnt.Content + "</b>"
			}
		}
		font = append(font, fnt)
	}
	return font
}

var formCtrlPresets = map[FormControlType]formCtrlPreset{
	FormControlNote: {
		objectType:   "Note",
		autoFill:     "True",
		filled:       "",
		fillColor:    "#FBF6D6",
		stroked:      "",
		strokeColor:  "#EDEAA1",
		strokeButton: "",
		fill: &vFill{
			Color2: "#FBFE82",
			Angle:  -180,
			Type:   "gradient",
			Fill:   &oFill{Ext: "view", Type: "gradientUnscaled"},
		},
		textHAlign:  "",
		textVAlign:  "",
		noThreeD:    nil,
		firstButton: nil,
		shadow:      &vShadow{On: "t", Color: "black", Obscured: "t"},
	},
	FormControlButton: {
		objectType:   "Button",
		autoFill:     "True",
		filled:       "",
		fillColor:    "buttonFace [67]",
		stroked:      "",
		strokeColor:  "windowText [64]",
		strokeButton: "t",
		fill: &vFill{
			Color2: "buttonFace [67]",
			Angle:  -180,
			Type:   "gradient",
			Fill:   &oFill{Ext: "view", Type: "gradientUnscaled"},
		},
		textHAlign:  "Center",
		textVAlign:  "Center",
		noThreeD:    nil,
		firstButton: nil,
		shadow:      nil,
	},
	FormControlCheckBox: {
		objectType:   "Checkbox",
		autoFill:     "True",
		filled:       "f",
		fillColor:    "window [65]",
		stroked:      "f",
		strokeColor:  "windowText [64]",
		strokeButton: "",
		fill:         nil,
		textHAlign:   "",
		textVAlign:   "Center",
		noThreeD:     stringPtr(""),
		firstButton:  nil,
		shadow:       nil,
	},
	FormControlGroupBox: {
		objectType:   "GBox",
		autoFill:     "False",
		filled:       "f",
		fillColor:    "",
		stroked:      "f",
		strokeColor:  "windowText [64]",
		strokeButton: "",
		fill:         nil,
		textHAlign:   "",
		textVAlign:   "",
		noThreeD:     stringPtr(""),
		firstButton:  nil,
		shadow:       nil,
	},
	FormControlLabel: {
		objectType:   "Label",
		autoFill:     "False",
		filled:       "f",
		fillColor:    "window [65]",
		stroked:      "f",
		strokeColor:  "windowText [64]",
		strokeButton: "",
		fill:         nil,
		textHAlign:   "",
		textVAlign:   "",
		noThreeD:     nil,
		firstButton:  nil,
		shadow:       nil,
	},
	FormControlOptionButton: {
		objectType:   "Radio",
		autoFill:     "False",
		filled:       "f",
		fillColor:    "window [65]",
		stroked:      "f",
		strokeColor:  "windowText [64]",
		strokeButton: "",
		fill:         nil,
		textHAlign:   "",
		textVAlign:   "Center",
		noThreeD:     stringPtr(""),
		firstButton:  stringPtr(""),
		shadow:       nil,
	},
	FormControlScrollBar: {
		objectType:   "Scroll",
		autoFill:     "",
		filled:       "",
		fillColor:    "",
		stroked:      "f",
		strokeColor:  "windowText [64]",
		strokeButton: "",
		fill:         nil,
		textHAlign:   "",
		textVAlign:   "",
		noThreeD:     nil,
		firstButton:  nil,
		shadow:       nil,
	},
	FormControlSpinButton: {
		objectType:   "Spin",
		autoFill:     "False",
		filled:       "",
		fillColor:    "",
		stroked:      "f",
		strokeColor:  "windowText [64]",
		strokeButton: "",
		fill:         nil,
		textHAlign:   "",
		textVAlign:   "",
		noThreeD:     nil,
		firstButton:  nil,
		shadow:       nil,
	},
}

// addFormCtrl check and add scroll bar or spinner form control by given options.
func (sp *encodeShape) addFormCtrl(opts *vmlOptions) error {
	if opts.Type != FormControlScrollBar && opts.Type != FormControlSpinButton {
		return nil
	}
	if opts.CurrentVal > MaxFormControlValue ||
		opts.MinVal > MaxFormControlValue ||
		opts.MaxVal > MaxFormControlValue ||
		opts.IncChange > MaxFormControlValue ||
		opts.PageChange > MaxFormControlValue {
		return ErrFormControlValue
	}
	if opts.CellLink != "" {
		if _, _, err := CellNameToCoordinates(opts.CellLink); err != nil {
			return err
		}
	}
	sp.ClientData.FmlaLink = opts.CellLink
	sp.ClientData.Val = opts.CurrentVal
	sp.ClientData.Min = opts.MinVal
	sp.ClientData.Max = opts.MaxVal
	sp.ClientData.Inc = opts.IncChange
	sp.ClientData.Page = opts.PageChange
	if opts.Type == FormControlScrollBar {
		if opts.Horizontally {
			sp.ClientData.Horiz = stringPtr("")
		}
		sp.ClientData.Dx = 15
	}
	return nil
}

// addFormCtrlShape returns a VML shape by given preset and options.
func (f *File) addFormCtrlShape(preset formCtrlPreset, col, row int, anchor string, opts *vmlOptions) (*encodeShape, error) {
	sp := encodeShape{
		Fill:   preset.fill,
		Shadow: preset.shadow,
		Path:   &vPath{ConnectType: "none"},
		TextBox: &vTextBox{
			Style: "mso-direction-alt:auto",
			Div:   &xlsxDiv{Style: "text-align:left"},
		},
		ClientData: &xClientData{
			ObjectType:  preset.objectType,
			Anchor:      anchor,
			AutoFill:    preset.autoFill,
			Row:         intPtr(row - 1),
			Column:      intPtr(col - 1),
			TextHAlign:  preset.textHAlign,
			TextVAlign:  preset.textVAlign,
			NoThreeD:    preset.noThreeD,
			FirstButton: preset.firstButton,
		},
	}
	if opts.Format.PrintObject != nil && !*opts.Format.PrintObject {
		sp.ClientData.PrintObject = "False"
	}
	if opts.Format.Positioning != "" {
		idx := inStrSlice(supportedPositioning, opts.Format.Positioning, true)
		if idx == -1 {
			return &sp, ErrParameterInvalid
		}
		sp.ClientData.MoveWithCells = []*string{stringPtr(""), nil, nil}[idx]
		sp.ClientData.SizeWithCells = []*string{stringPtr(""), stringPtr(""), nil}[idx]
	}
	if opts.FormControl.Type == FormControlNote {
		sp.ClientData.MoveWithCells = stringPtr("")
		sp.ClientData.SizeWithCells = stringPtr("")
	}
	if !opts.formCtrl {
		return &sp, nil
	}
	sp.TextBox.Div.Font = formCtrlText(opts)
	sp.ClientData.FmlaMacro = opts.Macro
	if (opts.Type == FormControlCheckBox || opts.Type == FormControlOptionButton) && opts.Checked {
		sp.ClientData.Checked = 1
	}
	return &sp, sp.addFormCtrl(opts)
}

// addDrawingVML provides a function to create VML drawing XML as
// xl/drawings/vmlDrawing%d.vml by given data ID, XML path and VML options. The
// anchor value is a comma-separated list of data written out as: LeftColumn,
// LeftOffset, TopRow, TopOffset, RightColumn, RightOffset, BottomRow,
// BottomOffset.
func (f *File) addDrawingVML(dataID int, drawingVML string, opts *vmlOptions) error {
	col, row, err := CellNameToCoordinates(opts.FormControl.Cell)
	if err != nil {
		return err
	}
	anchor := fmt.Sprintf("%d, 23, %d, 0, %d, %d, %d, 5", col, row, col+opts.rows+2, col+opts.cols-1, row+opts.rows+2)
	vmlID, vml, preset := 202, f.VMLDrawing[drawingVML], formCtrlPresets[opts.Type]
	style := "position:absolute;73.5pt;width:108pt;height:59.25pt;z-index:1;visibility:hidden"
	if opts.formCtrl {
		vmlID = 201
		style = "position:absolute;73.5pt;width:108pt;height:59.25pt;z-index:1;mso-wrap-style:tight"
		colStart, rowStart, colEnd, rowEnd, x2, y2 := f.positionObjectPixels(opts.sheet, col, row, opts.Format.OffsetX, opts.Format.OffsetY, int(opts.Width), int(opts.Height))
		anchor = fmt.Sprintf("%d, 0, %d, 0, %d, %d, %d, %d", colStart, rowStart, colEnd, x2, rowEnd, y2)
	}
	if vml == nil {
		vml = &vmlDrawing{
			XMLNSv:  "urn:schemas-microsoft-com:vml",
			XMLNSo:  "urn:schemas-microsoft-com:office:office",
			XMLNSx:  "urn:schemas-microsoft-com:office:excel",
			XMLNSmv: "http://macVmlSchemaUri",
			ShapeLayout: &xlsxShapeLayout{
				Ext: "edit", IDmap: &xlsxIDmap{Ext: "edit", Data: dataID},
			},
			ShapeType: &xlsxShapeType{
				ID:        fmt.Sprintf("_x0000_t%d", vmlID),
				CoordSize: "21600,21600",
				Spt:       202,
				Path:      "m0,0l0,21600,21600,21600,21600,0xe",
				Stroke:    &xlsxStroke{JoinStyle: "miter"},
				VPath:     &vPath{GradientShapeOK: "t", ConnectType: "rect"},
			},
		}
		// Load exist VML shapes from xl/drawings/vmlDrawing%d.vml
		d, err := f.decodeVMLDrawingReader(drawingVML)
		if err != nil {
			return err
		}
		if d != nil {
			vml.ShapeType.ID = d.ShapeType.ID
			vml.ShapeType.CoordSize = d.ShapeType.CoordSize
			vml.ShapeType.Spt = d.ShapeType.Spt
			vml.ShapeType.Path = d.ShapeType.Path
			for _, v := range d.Shape {
				s := xlsxShape{
					ID:          v.ID,
					Type:        v.Type,
					Style:       v.Style,
					Button:      v.Button,
					Filled:      v.Filled,
					FillColor:   v.FillColor,
					InsetMode:   v.InsetMode,
					Stroked:     v.Stroked,
					StrokeColor: v.StrokeColor,
					Val:         v.Val,
				}
				vml.Shape = append(vml.Shape, s)
			}
		}
	}
	sp, err := f.addFormCtrlShape(preset, col, row, anchor, opts)
	if err != nil {
		return err
	}
	s, _ := xml.Marshal(sp)
	shape := xlsxShape{
		ID:          "_x0000_s1025",
		Type:        fmt.Sprintf("#_x0000_t%d", vmlID),
		Style:       style,
		Button:      preset.strokeButton,
		Filled:      preset.filled,
		FillColor:   preset.fillColor,
		Stroked:     preset.stroked,
		StrokeColor: preset.strokeColor,
		Val:         string(s[13 : len(s)-14]),
	}
	vml.Shape = append(vml.Shape, shape)
	f.VMLDrawing[drawingVML] = vml
	return err
}

// GetFormControls retrieves all form controls in a worksheet by a given
// worksheet name. Note that, this function does not support getting the width
// and height of the form controls currently.
func (f *File) GetFormControls(sheet string) ([]FormControl, error) {
	var formControls []FormControl
	// Read sheet data
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return formControls, err
	}
	if ws.LegacyDrawing == nil {
		return formControls, err
	}
	target := f.getSheetRelationshipsTargetByID(sheet, ws.LegacyDrawing.RID)
	drawingVML := strings.ReplaceAll(target, "..", "xl")
	vml := f.VMLDrawing[drawingVML]
	if vml == nil {
		// Load exist VML shapes from xl/drawings/vmlDrawing%d.vml
		d, err := f.decodeVMLDrawingReader(drawingVML)
		if err != nil {
			return formControls, err
		}
		for _, sp := range d.Shape {
			if sp.Type != "#_x0000_t201" {
				continue
			}
			formControl, err := extractFormControl(sp.Val)
			if err != nil {
				return formControls, err
			}
			if formControl.Type == FormControlNote || formControl.Cell == "" {
				continue
			}
			formControls = append(formControls, formControl)
		}
		return formControls, err
	}
	for _, sp := range vml.Shape {
		if sp.Type != "#_x0000_t201" {
			continue
		}
		formControl, err := extractFormControl(sp.Val)
		if err != nil {
			return formControls, err
		}
		if formControl.Type == FormControlNote || formControl.Cell == "" {
			continue
		}
		formControls = append(formControls, formControl)
	}
	return formControls, err
}

// extractFormControl provides a function to extract form controls for a
// worksheets by given client data.
func extractFormControl(clientData string) (FormControl, error) {
	var (
		err         error
		formControl FormControl
		shapeVal    decodeShapeVal
	)
	if err = xml.Unmarshal([]byte(fmt.Sprintf("<shape>%s</shape>", clientData)), &shapeVal); err != nil {
		return formControl, err
	}
	for formCtrlType, preset := range formCtrlPresets {
		if shapeVal.ClientData.ObjectType == preset.objectType && shapeVal.ClientData.Anchor != "" {
			formControl.Paragraph = extractVMLFont(shapeVal.TextBox.Div.Font)
			if len(formControl.Paragraph) > 0 && formControl.Paragraph[0].Font == nil {
				formControl.Text = formControl.Paragraph[0].Text
				formControl.Paragraph = formControl.Paragraph[1:]
			}
			formControl.Type = formCtrlType
			col, row, err := extractAnchorCell(shapeVal.ClientData.Anchor)
			if err != nil {
				return formControl, err
			}
			if formControl.Cell, err = CoordinatesToCellName(col+1, row+1); err != nil {
				return formControl, err
			}
			formControl.Macro = shapeVal.ClientData.FmlaMacro
			formControl.Checked = shapeVal.ClientData.Checked != 0
			formControl.CellLink = shapeVal.ClientData.FmlaLink
			formControl.CurrentVal = shapeVal.ClientData.Val
			formControl.MinVal = shapeVal.ClientData.Min
			formControl.MaxVal = shapeVal.ClientData.Max
			formControl.IncChange = shapeVal.ClientData.Inc
			formControl.PageChange = shapeVal.ClientData.Page
			formControl.Horizontally = shapeVal.ClientData.Horiz != nil
		}
	}
	return formControl, err
}

// extractAnchorCell extract left-top cell coordinates from given VML anchor
// comma-separated list values.
func extractAnchorCell(anchor string) (int, int, error) {
	var (
		leftCol, topRow int
		err             error
		pos             = strings.Split(anchor, ",")
	)
	if len(pos) != 8 {
		return leftCol, topRow, ErrParameterInvalid
	}
	leftCol, err = strconv.Atoi(strings.TrimSpace(pos[0]))
	if err != nil {
		return leftCol, topRow, ErrColumnNumber
	}
	topRow, err = strconv.Atoi(strings.TrimSpace(pos[2]))
	return leftCol, topRow, err
}

// extractVMLFont extract rich-text and font format from given VML font element.
func extractVMLFont(font []decodeVMLFont) []RichTextRun {
	var runs []RichTextRun
	extractU := func(u *decodeVMLFontU, run *RichTextRun) {
		if u == nil {
			return
		}
		run.Text += u.Val
		if run.Font == nil {
			run.Font = &Font{}
		}
		run.Font.Underline = "single"
		if u.Class == "font1" {
			run.Font.Underline = "double"
		}
	}
	extractI := func(i *decodeVMLFontI, run *RichTextRun) {
		if i == nil {
			return
		}
		extractU(i.U, run)
		run.Text += i.Val
		if run.Font == nil {
			run.Font = &Font{}
		}
		run.Font.Italic = true
	}
	extractB := func(b *decodeVMLFontB, run *RichTextRun) {
		if b == nil {
			return
		}
		extractI(b.I, run)
		run.Text += b.Val
		if run.Font == nil {
			run.Font = &Font{}
		}
		run.Font.Bold = true
	}
	for _, fnt := range font {
		var run RichTextRun
		extractB(fnt.B, &run)
		extractI(fnt.I, &run)
		extractU(fnt.U, &run)
		run.Text += fnt.Val
		if fnt.Face != "" || fnt.Size > 0 || fnt.Color != "" {
			if run.Font == nil {
				run.Font = &Font{}
			}
			run.Font.Family = fnt.Face
			run.Font.Size = float64(fnt.Size / 20)
			run.Font.Color = fnt.Color
		}
		runs = append(runs, run)
	}
	return runs
}
