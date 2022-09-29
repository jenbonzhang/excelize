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
	"io"
	"strings"
)

// addSparklineGroupByStyle provides a function to create x14:sparklineGroups
// element by given sparkline style ID.
func (f *File) addSparklineGroupByStyle(ID int) *xlsxX14SparklineGroup {
	groups := []*xlsxX14SparklineGroup{
		{
			ColorSeries:   &xlsxTabColor{Theme: 4, Tint: -0.499984740745262},
			ColorNegative: &xlsxTabColor{Theme: 5},
			ColorMarkers:  &xlsxTabColor{Theme: 4, Tint: -0.499984740745262},
			ColorFirst:    &xlsxTabColor{Theme: 4, Tint: 0.39997558519241921},
			ColorLast:     &xlsxTabColor{Theme: 4, Tint: 0.39997558519241921},
			ColorHigh:     &xlsxTabColor{Theme: 4},
			ColorLow:      &xlsxTabColor{Theme: 4},
		}, // 0
		{
			ColorSeries:   &xlsxTabColor{Theme: 4, Tint: -0.499984740745262},
			ColorNegative: &xlsxTabColor{Theme: 5},
			ColorMarkers:  &xlsxTabColor{Theme: 4, Tint: -0.499984740745262},
			ColorFirst:    &xlsxTabColor{Theme: 4, Tint: 0.39997558519241921},
			ColorLast:     &xlsxTabColor{Theme: 4, Tint: 0.39997558519241921},
			ColorHigh:     &xlsxTabColor{Theme: 4},
			ColorLow:      &xlsxTabColor{Theme: 4},
		}, // 1
		{
			ColorSeries:   &xlsxTabColor{Theme: 5, Tint: -0.499984740745262},
			ColorNegative: &xlsxTabColor{Theme: 6},
			ColorMarkers:  &xlsxTabColor{Theme: 5, Tint: -0.499984740745262},
			ColorFirst:    &xlsxTabColor{Theme: 5, Tint: 0.39997558519241921},
			ColorLast:     &xlsxTabColor{Theme: 5, Tint: 0.39997558519241921},
			ColorHigh:     &xlsxTabColor{Theme: 5},
			ColorLow:      &xlsxTabColor{Theme: 5},
		}, // 2
		{
			ColorSeries:   &xlsxTabColor{Theme: 6, Tint: -0.499984740745262},
			ColorNegative: &xlsxTabColor{Theme: 7},
			ColorMarkers:  &xlsxTabColor{Theme: 6, Tint: -0.499984740745262},
			ColorFirst:    &xlsxTabColor{Theme: 6, Tint: 0.39997558519241921},
			ColorLast:     &xlsxTabColor{Theme: 6, Tint: 0.39997558519241921},
			ColorHigh:     &xlsxTabColor{Theme: 6},
			ColorLow:      &xlsxTabColor{Theme: 6},
		}, // 3
		{
			ColorSeries:   &xlsxTabColor{Theme: 7, Tint: -0.499984740745262},
			ColorNegative: &xlsxTabColor{Theme: 8},
			ColorMarkers:  &xlsxTabColor{Theme: 7, Tint: -0.499984740745262},
			ColorFirst:    &xlsxTabColor{Theme: 7, Tint: 0.39997558519241921},
			ColorLast:     &xlsxTabColor{Theme: 7, Tint: 0.39997558519241921},
			ColorHigh:     &xlsxTabColor{Theme: 7},
			ColorLow:      &xlsxTabColor{Theme: 7},
		}, // 4
		{
			ColorSeries:   &xlsxTabColor{Theme: 8, Tint: -0.499984740745262},
			ColorNegative: &xlsxTabColor{Theme: 9},
			ColorMarkers:  &xlsxTabColor{Theme: 8, Tint: -0.499984740745262},
			ColorFirst:    &xlsxTabColor{Theme: 8, Tint: 0.39997558519241921},
			ColorLast:     &xlsxTabColor{Theme: 8, Tint: 0.39997558519241921},
			ColorHigh:     &xlsxTabColor{Theme: 8},
			ColorLow:      &xlsxTabColor{Theme: 8},
		}, // 5
		{
			ColorSeries:   &xlsxTabColor{Theme: 9, Tint: -0.499984740745262},
			ColorNegative: &xlsxTabColor{Theme: 4},
			ColorMarkers:  &xlsxTabColor{Theme: 9, Tint: -0.499984740745262},
			ColorFirst:    &xlsxTabColor{Theme: 9, Tint: 0.39997558519241921},
			ColorLast:     &xlsxTabColor{Theme: 9, Tint: 0.39997558519241921},
			ColorHigh:     &xlsxTabColor{Theme: 9},
			ColorLow:      &xlsxTabColor{Theme: 9},
		}, // 6
		{
			ColorSeries:   &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorNegative: &xlsxTabColor{Theme: 5},
			ColorMarkers:  &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 5},
			ColorLow:      &xlsxTabColor{Theme: 5},
		}, // 7
		{
			ColorSeries:   &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorNegative: &xlsxTabColor{Theme: 6},
			ColorMarkers:  &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
		}, // 8
		{
			ColorSeries:   &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorNegative: &xlsxTabColor{Theme: 7},
			ColorMarkers:  &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
		}, // 9
		{
			ColorSeries:   &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorNegative: &xlsxTabColor{Theme: 8},
			ColorMarkers:  &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
		}, // 10
		{
			ColorSeries:   &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorNegative: &xlsxTabColor{Theme: 9},
			ColorMarkers:  &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
		}, // 11
		{
			ColorSeries:   &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorNegative: &xlsxTabColor{Theme: 4},
			ColorMarkers:  &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
		}, // 12
		{
			ColorSeries:   &xlsxTabColor{Theme: 4},
			ColorNegative: &xlsxTabColor{Theme: 5},
			ColorMarkers:  &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
		}, // 13
		{
			ColorSeries:   &xlsxTabColor{Theme: 5},
			ColorNegative: &xlsxTabColor{Theme: 6},
			ColorMarkers:  &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
		}, // 14
		{
			ColorSeries:   &xlsxTabColor{Theme: 6},
			ColorNegative: &xlsxTabColor{Theme: 7},
			ColorMarkers:  &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
		}, // 15
		{
			ColorSeries:   &xlsxTabColor{Theme: 7},
			ColorNegative: &xlsxTabColor{Theme: 8},
			ColorMarkers:  &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
		}, // 16
		{
			ColorSeries:   &xlsxTabColor{Theme: 8},
			ColorNegative: &xlsxTabColor{Theme: 9},
			ColorMarkers:  &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
		}, // 17
		{
			ColorSeries:   &xlsxTabColor{Theme: 9},
			ColorNegative: &xlsxTabColor{Theme: 4},
			ColorMarkers:  &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
		}, // 18
		{
			ColorSeries:   &xlsxTabColor{Theme: 4, Tint: 0.39997558519241921},
			ColorNegative: &xlsxTabColor{Theme: 0, Tint: -0.499984740745262},
			ColorMarkers:  &xlsxTabColor{Theme: 4, Tint: 0.79998168889431442},
			ColorFirst:    &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 4, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 4, Tint: -0.499984740745262},
			ColorLow:      &xlsxTabColor{Theme: 4, Tint: -0.499984740745262},
		}, // 19
		{
			ColorSeries:   &xlsxTabColor{Theme: 5, Tint: 0.39997558519241921},
			ColorNegative: &xlsxTabColor{Theme: 0, Tint: -0.499984740745262},
			ColorMarkers:  &xlsxTabColor{Theme: 5, Tint: 0.79998168889431442},
			ColorFirst:    &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 5, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 5, Tint: -0.499984740745262},
			ColorLow:      &xlsxTabColor{Theme: 5, Tint: -0.499984740745262},
		}, // 20
		{
			ColorSeries:   &xlsxTabColor{Theme: 6, Tint: 0.39997558519241921},
			ColorNegative: &xlsxTabColor{Theme: 0, Tint: -0.499984740745262},
			ColorMarkers:  &xlsxTabColor{Theme: 6, Tint: 0.79998168889431442},
			ColorFirst:    &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 6, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 6, Tint: -0.499984740745262},
			ColorLow:      &xlsxTabColor{Theme: 6, Tint: -0.499984740745262},
		}, // 21
		{
			ColorSeries:   &xlsxTabColor{Theme: 7, Tint: 0.39997558519241921},
			ColorNegative: &xlsxTabColor{Theme: 0, Tint: -0.499984740745262},
			ColorMarkers:  &xlsxTabColor{Theme: 7, Tint: 0.79998168889431442},
			ColorFirst:    &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 7, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 7, Tint: -0.499984740745262},
			ColorLow:      &xlsxTabColor{Theme: 7, Tint: -0.499984740745262},
		}, // 22
		{
			ColorSeries:   &xlsxTabColor{Theme: 8, Tint: 0.39997558519241921},
			ColorNegative: &xlsxTabColor{Theme: 0, Tint: -0.499984740745262},
			ColorMarkers:  &xlsxTabColor{Theme: 8, Tint: 0.79998168889431442},
			ColorFirst:    &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 8, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 8, Tint: -0.499984740745262},
			ColorLow:      &xlsxTabColor{Theme: 8, Tint: -0.499984740745262},
		}, // 23
		{
			ColorSeries:   &xlsxTabColor{Theme: 9, Tint: 0.39997558519241921},
			ColorNegative: &xlsxTabColor{Theme: 0, Tint: -0.499984740745262},
			ColorMarkers:  &xlsxTabColor{Theme: 9, Tint: 0.79998168889431442},
			ColorFirst:    &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 9, Tint: -0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 9, Tint: -0.499984740745262},
			ColorLow:      &xlsxTabColor{Theme: 9, Tint: -0.499984740745262},
		}, // 24
		{
			ColorSeries:   &xlsxTabColor{Theme: 1, Tint: 0.499984740745262},
			ColorNegative: &xlsxTabColor{Theme: 1, Tint: 0.249977111117893},
			ColorMarkers:  &xlsxTabColor{Theme: 1, Tint: 0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 1, Tint: 0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 1, Tint: 0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 1, Tint: 0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 1, Tint: 0.249977111117893},
		}, // 25
		{
			ColorSeries:   &xlsxTabColor{Theme: 1, Tint: 0.34998626667073579},
			ColorNegative: &xlsxTabColor{Theme: 0, Tint: 0.249977111117893},
			ColorMarkers:  &xlsxTabColor{Theme: 0, Tint: 0.249977111117893},
			ColorFirst:    &xlsxTabColor{Theme: 0, Tint: 0.249977111117893},
			ColorLast:     &xlsxTabColor{Theme: 0, Tint: 0.249977111117893},
			ColorHigh:     &xlsxTabColor{Theme: 0, Tint: 0.249977111117893},
			ColorLow:      &xlsxTabColor{Theme: 0, Tint: 0.249977111117893},
		}, // 26
		{
			ColorSeries:   &xlsxTabColor{RGB: "FF323232"},
			ColorNegative: &xlsxTabColor{RGB: "FFD00000"},
			ColorMarkers:  &xlsxTabColor{RGB: "FFD00000"},
			ColorFirst:    &xlsxTabColor{RGB: "FFD00000"},
			ColorLast:     &xlsxTabColor{RGB: "FFD00000"},
			ColorHigh:     &xlsxTabColor{RGB: "FFD00000"},
			ColorLow:      &xlsxTabColor{RGB: "FFD00000"},
		}, // 27
		{
			ColorSeries:   &xlsxTabColor{RGB: "FF000000"},
			ColorNegative: &xlsxTabColor{RGB: "FF0070C0"},
			ColorMarkers:  &xlsxTabColor{RGB: "FF0070C0"},
			ColorFirst:    &xlsxTabColor{RGB: "FF0070C0"},
			ColorLast:     &xlsxTabColor{RGB: "FF0070C0"},
			ColorHigh:     &xlsxTabColor{RGB: "FF0070C0"},
			ColorLow:      &xlsxTabColor{RGB: "FF0070C0"},
		}, // 28
		{
			ColorSeries:   &xlsxTabColor{RGB: "FF376092"},
			ColorNegative: &xlsxTabColor{RGB: "FFD00000"},
			ColorMarkers:  &xlsxTabColor{RGB: "FFD00000"},
			ColorFirst:    &xlsxTabColor{RGB: "FFD00000"},
			ColorLast:     &xlsxTabColor{RGB: "FFD00000"},
			ColorHigh:     &xlsxTabColor{RGB: "FFD00000"},
			ColorLow:      &xlsxTabColor{RGB: "FFD00000"},
		}, // 29
		{
			ColorSeries:   &xlsxTabColor{RGB: "FF0070C0"},
			ColorNegative: &xlsxTabColor{RGB: "FF000000"},
			ColorMarkers:  &xlsxTabColor{RGB: "FF000000"},
			ColorFirst:    &xlsxTabColor{RGB: "FF000000"},
			ColorLast:     &xlsxTabColor{RGB: "FF000000"},
			ColorHigh:     &xlsxTabColor{RGB: "FF000000"},
			ColorLow:      &xlsxTabColor{RGB: "FF000000"},
		}, // 30
		{
			ColorSeries:   &xlsxTabColor{RGB: "FF5F5F5F"},
			ColorNegative: &xlsxTabColor{RGB: "FFFFB620"},
			ColorMarkers:  &xlsxTabColor{RGB: "FFD70077"},
			ColorFirst:    &xlsxTabColor{RGB: "FF5687C2"},
			ColorLast:     &xlsxTabColor{RGB: "FF359CEB"},
			ColorHigh:     &xlsxTabColor{RGB: "FF56BE79"},
			ColorLow:      &xlsxTabColor{RGB: "FFFF5055"},
		}, // 31
		{
			ColorSeries:   &xlsxTabColor{RGB: "FF5687C2"},
			ColorNegative: &xlsxTabColor{RGB: "FFFFB620"},
			ColorMarkers:  &xlsxTabColor{RGB: "FFD70077"},
			ColorFirst:    &xlsxTabColor{RGB: "FF777777"},
			ColorLast:     &xlsxTabColor{RGB: "FF359CEB"},
			ColorHigh:     &xlsxTabColor{RGB: "FF56BE79"},
			ColorLow:      &xlsxTabColor{RGB: "FFFF5055"},
		}, // 32
		{
			ColorSeries:   &xlsxTabColor{RGB: "FFC6EFCE"},
			ColorNegative: &xlsxTabColor{RGB: "FFFFC7CE"},
			ColorMarkers:  &xlsxTabColor{RGB: "FF8CADD6"},
			ColorFirst:    &xlsxTabColor{RGB: "FFFFDC47"},
			ColorLast:     &xlsxTabColor{RGB: "FFFFEB9C"},
			ColorHigh:     &xlsxTabColor{RGB: "FF60D276"},
			ColorLow:      &xlsxTabColor{RGB: "FFFF5367"},
		}, // 33
		{
			ColorSeries:   &xlsxTabColor{RGB: "FF00B050"},
			ColorNegative: &xlsxTabColor{RGB: "FFFF0000"},
			ColorMarkers:  &xlsxTabColor{RGB: "FF0070C0"},
			ColorFirst:    &xlsxTabColor{RGB: "FFFFC000"},
			ColorLast:     &xlsxTabColor{RGB: "FFFFC000"},
			ColorHigh:     &xlsxTabColor{RGB: "FF00B050"},
			ColorLow:      &xlsxTabColor{RGB: "FFFF0000"},
		}, // 34
		{
			ColorSeries:   &xlsxTabColor{Theme: 3},
			ColorNegative: &xlsxTabColor{Theme: 9},
			ColorMarkers:  &xlsxTabColor{Theme: 8},
			ColorFirst:    &xlsxTabColor{Theme: 4},
			ColorLast:     &xlsxTabColor{Theme: 5},
			ColorHigh:     &xlsxTabColor{Theme: 6},
			ColorLow:      &xlsxTabColor{Theme: 7},
		}, // 35
		{
			ColorSeries:   &xlsxTabColor{Theme: 1},
			ColorNegative: &xlsxTabColor{Theme: 9},
			ColorMarkers:  &xlsxTabColor{Theme: 8},
			ColorFirst:    &xlsxTabColor{Theme: 4},
			ColorLast:     &xlsxTabColor{Theme: 5},
			ColorHigh:     &xlsxTabColor{Theme: 6},
			ColorLow:      &xlsxTabColor{Theme: 7},
		}, // 36
	}
	return groups[ID]
}

// AddSparkline provides a function to add sparklines to the worksheet by
// given formatting options. Sparklines are small charts that fit in a single
// cell and are used to show trends in data. Sparklines are a feature of Excel
// 2010 and later only. You can write them to an XLSX file that can be read by
// Excel 2007, but they won't be displayed. For example, add a grouped
// sparkline. Changes are applied to all three:
//
//	err := f.AddSparkline("Sheet1", &excelize.SparklineOption{
//	    Location: []string{"A1", "A2", "A3"},
//	    Range:    []string{"Sheet2!A1:J1", "Sheet2!A2:J2", "Sheet2!A3:J3"},
//	    Markers:  true,
//	})
//
// The following shows the formatting options of sparkline supported by excelize:
//
//	 Parameter | Description
//	-----------+--------------------------------------------
//	 Location  | Required, must have the same number with 'Range' parameter
//	 Range     | Required, must have the same number with 'Location' parameter
//	 Type      | Enumeration value: line, column, win_loss
//	 Style     | Value range: 0 - 35
//	 Hight     | Toggle sparkline high points
//	 Low       | Toggle sparkline low points
//	 First     | Toggle sparkline first points
//	 Last      | Toggle sparkline last points
//	 Negative  | Toggle sparkline negative points
//	 Markers   | Toggle sparkline markers
//	 ColorAxis | An RGB Color is specified as RRGGBB
//	 Axis      | Show sparkline axis
func (f *File) AddSparkline(sheet string, opts *SparklineOptions) error {
	var (
		err                            error
		ws                             *xlsxWorksheet
		sparkType                      string
		sparkTypes                     map[string]string
		specifiedSparkTypes            string
		ok                             bool
		group                          *xlsxX14SparklineGroup
		groups                         *xlsxX14SparklineGroups
		sparklineGroupsBytes, extBytes []byte
	)

	// parameter validation
	if ws, err = f.parseFormatAddSparklineSet(sheet, opts); err != nil {
		return err
	}
	// Handle the sparkline type
	sparkType = "line"
	sparkTypes = map[string]string{"line": "line", "column": "column", "win_loss": "stacked"}
	if opts.Type != "" {
		if specifiedSparkTypes, ok = sparkTypes[opts.Type]; !ok {
			err = ErrSparklineType
			return err
		}
		sparkType = specifiedSparkTypes
	}
	group = f.addSparklineGroupByStyle(opts.Style)
	group.Type = sparkType
	group.ColorAxis = &xlsxColor{RGB: "FF000000"}
	group.DisplayEmptyCellsAs = "gap"
	group.High = opts.High
	group.Low = opts.Low
	group.First = opts.First
	group.Last = opts.Last
	group.Negative = opts.Negative
	group.DisplayXAxis = opts.Axis
	group.Markers = opts.Markers
	if opts.SeriesColor != "" {
		group.ColorSeries = &xlsxTabColor{
			RGB: getPaletteColor(opts.SeriesColor),
		}
	}
	if opts.Reverse {
		group.RightToLeft = opts.Reverse
	}
	f.addSparkline(opts, group)
	if ws.ExtLst.Ext != "" { // append mode ext
		if err = f.appendSparkline(ws, group, groups); err != nil {
			return err
		}
	} else {
		groups = &xlsxX14SparklineGroups{
			XMLNSXM:         NameSpaceSpreadSheetExcel2006Main.Value,
			SparklineGroups: []*xlsxX14SparklineGroup{group},
		}
		if sparklineGroupsBytes, err = xml.Marshal(groups); err != nil {
			return err
		}
		if extBytes, err = xml.Marshal(&xlsxWorksheetExt{
			URI:     ExtURISparklineGroups,
			Content: string(sparklineGroupsBytes),
		}); err != nil {
			return err
		}
		ws.ExtLst.Ext = string(extBytes)
	}
	f.addSheetNameSpace(sheet, NameSpaceSpreadSheetX14)
	return err
}

// parseFormatAddSparklineSet provides a function to validate sparkline
// properties.
func (f *File) parseFormatAddSparklineSet(sheet string, opts *SparklineOptions) (*xlsxWorksheet, error) {
	ws, err := f.workSheetReader(sheet)
	if err != nil {
		return ws, err
	}
	if opts == nil {
		return ws, ErrParameterRequired
	}
	if len(opts.Location) < 1 {
		return ws, ErrSparklineLocation
	}
	if len(opts.Range) < 1 {
		return ws, ErrSparklineRange
	}
	// The range and locations must match
	if len(opts.Location) != len(opts.Range) {
		return ws, ErrSparkline
	}
	if opts.Style < 0 || opts.Style > 35 {
		return ws, ErrSparklineStyle
	}
	if ws.ExtLst == nil {
		ws.ExtLst = &xlsxExtLst{}
	}
	return ws, err
}

// addSparkline provides a function to create a sparkline in a sparkline group
// by given properties.
func (f *File) addSparkline(opts *SparklineOptions, group *xlsxX14SparklineGroup) {
	for idx, location := range opts.Location {
		group.Sparklines.Sparkline = append(group.Sparklines.Sparkline, &xlsxX14Sparkline{
			F:     opts.Range[idx],
			Sqref: location,
		})
	}
}

// appendSparkline provides a function to append sparkline to sparkline
// groups.
func (f *File) appendSparkline(ws *xlsxWorksheet, group *xlsxX14SparklineGroup, groups *xlsxX14SparklineGroups) error {
	var (
		err                                                    error
		idx                                                    int
		decodeExtLst                                           *decodeWorksheetExt
		decodeSparklineGroups                                  *decodeX14SparklineGroups
		ext                                                    *xlsxWorksheetExt
		sparklineGroupsBytes, sparklineGroupBytes, extLstBytes []byte
	)
	decodeExtLst = new(decodeWorksheetExt)
	if err = f.xmlNewDecoder(strings.NewReader("<extLst>" + ws.ExtLst.Ext + "</extLst>")).
		Decode(decodeExtLst); err != nil && err != io.EOF {
		return err
	}
	for idx, ext = range decodeExtLst.Ext {
		if ext.URI == ExtURISparklineGroups {
			decodeSparklineGroups = new(decodeX14SparklineGroups)
			if err = f.xmlNewDecoder(strings.NewReader(ext.Content)).
				Decode(decodeSparklineGroups); err != nil && err != io.EOF {
				return err
			}
			if sparklineGroupBytes, err = xml.Marshal(group); err != nil {
				return err
			}
			if groups == nil {
				groups = &xlsxX14SparklineGroups{}
			}
			groups.XMLNSXM = NameSpaceSpreadSheetExcel2006Main.Value
			groups.Content = decodeSparklineGroups.Content + string(sparklineGroupBytes)
			if sparklineGroupsBytes, err = xml.Marshal(groups); err != nil {
				return err
			}
			decodeExtLst.Ext[idx].Content = string(sparklineGroupsBytes)
		}
	}
	if extLstBytes, err = xml.Marshal(decodeExtLst); err != nil {
		return err
	}
	ws.ExtLst = &xlsxExtLst{
		Ext: strings.TrimSuffix(strings.TrimPrefix(string(extLstBytes), "<extLst>"), "</extLst>"),
	}
	return err
}
