// Copyright 2016 - 2020 The excelize Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// Package excelize providing a set of functions that allow you to write to
// and read from XLSX / XLSM / XLTM files. Supports reading and writing
// spreadsheet documents generated by Microsoft Exce™ 2007 and later. Supports
// complex components by high compatibility, and provided streaming API for
// generating or reading data from a worksheet with huge amounts of data. This
// library needs Go version 1.10 or later.

package excelize

import (
	"container/list"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/xuri/efp"
)

// Excel formula errors
const (
	formulaErrorDIV         = "#DIV/0!"
	formulaErrorNAME        = "#NAME?"
	formulaErrorNA          = "#N/A"
	formulaErrorNUM         = "#NUM!"
	formulaErrorVALUE       = "#VALUE!"
	formulaErrorREF         = "#REF!"
	formulaErrorNULL        = "#NULL"
	formulaErrorSPILL       = "#SPILL!"
	formulaErrorCALC        = "#CALC!"
	formulaErrorGETTINGDATA = "#GETTING_DATA"
)

// cellRef defines the structure of a cell reference
type cellRef struct {
	Col   int
	Row   int
	Sheet string
}

// cellRef defines the structure of a cell range
type cellRange struct {
	From cellRef
	To   cellRef
}

type formulaFuncs struct{}

// CalcCellValue provides a function to get calculated cell value. This
// feature is currently in beta. Array formula, table formula and some other
// formulas are not supported currently.
func (f *File) CalcCellValue(sheet, cell string) (result string, err error) {
	var (
		formula string
		token   efp.Token
	)
	if formula, err = f.GetCellFormula(sheet, cell); err != nil {
		return
	}
	ps := efp.ExcelParser()
	tokens := ps.Parse(formula)
	if tokens == nil {
		return
	}
	if token, err = f.evalInfixExp(sheet, tokens); err != nil {
		return
	}
	result = token.TValue
	return
}

// getPriority calculate arithmetic operator priority.
func getPriority(token efp.Token) (pri int) {
	var priority = map[string]int{
		"*": 2,
		"/": 2,
		"+": 1,
		"-": 1,
	}
	pri, _ = priority[token.TValue]
	if token.TValue == "-" && token.TType == efp.TokenTypeOperatorPrefix {
		pri = 3
	}
	if token.TSubType == efp.TokenSubTypeStart && token.TType == efp.TokenTypeSubexpression { // (
		pri = 0
	}
	return
}

// evalInfixExp evaluate syntax analysis by given infix expression after
// lexical analysis. Evaluate an infix expression containing formulas by
// stacks:
//
//    opd  - Operand
//    opt  - Operator
//    opf  - Operation formula
//    opfd - Operand of the operation formula
//    opft - Operator of the operation formula
//
// Evaluate arguments of the operation formula by list:
//
//    args - Arguments of the operation formula
//
// TODO: handle subtypes: Nothing, Text, Logical, Error, Concatenation, Intersection, Union
//
func (f *File) evalInfixExp(sheet string, tokens []efp.Token) (efp.Token, error) {
	var err error
	opdStack, optStack, opfStack, opfdStack, opftStack := NewStack(), NewStack(), NewStack(), NewStack(), NewStack()
	argsList := list.New()
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		// out of function stack
		if opfStack.Len() == 0 {
			if err = f.parseToken(sheet, token, opdStack, optStack); err != nil {
				return efp.Token{}, err
			}
		}

		// function start
		if token.TType == efp.TokenTypeFunction && token.TSubType == efp.TokenSubTypeStart {
			opfStack.Push(token)
			continue
		}

		// in function stack, walk 2 token at once
		if opfStack.Len() > 0 {
			var nextToken efp.Token
			if i+1 < len(tokens) {
				nextToken = tokens[i+1]
			}

			// current token is args or range, skip next token, order required: parse reference first
			if token.TSubType == efp.TokenSubTypeRange {
				if !opftStack.Empty() {
					// parse reference: must reference at here
					result, err := f.parseReference(sheet, token.TValue)
					if err != nil {
						return efp.Token{TValue: formulaErrorNAME}, err
					}
					if len(result) != 1 {
						return efp.Token{}, errors.New(formulaErrorVALUE)
					}
					opfdStack.Push(efp.Token{
						TType:    efp.TokenTypeOperand,
						TSubType: efp.TokenSubTypeNumber,
						TValue:   result[0],
					})
					continue
				}
				if nextToken.TType == efp.TokenTypeArgument || nextToken.TType == efp.TokenTypeFunction {
					// parse reference: reference or range at here
					result, err := f.parseReference(sheet, token.TValue)
					if err != nil {
						return efp.Token{TValue: formulaErrorNAME}, err
					}
					for _, val := range result {
						argsList.PushBack(efp.Token{
							TType:    efp.TokenTypeOperand,
							TSubType: efp.TokenSubTypeNumber,
							TValue:   val,
						})
					}
					if len(result) == 0 {
						return efp.Token{}, errors.New(formulaErrorVALUE)
					}
					continue
				}
			}

			// check current token is opft
			if err = f.parseToken(sheet, token, opfdStack, opftStack); err != nil {
				return efp.Token{}, err
			}

			// current token is arg
			if token.TType == efp.TokenTypeArgument {
				for !opftStack.Empty() {
					// calculate trigger
					topOpt := opftStack.Peek().(efp.Token)
					if err := calculate(opfdStack, topOpt); err != nil {
						return efp.Token{}, err
					}
					opftStack.Pop()
				}
				if !opfdStack.Empty() {
					argsList.PushBack(opfdStack.Pop())
				}
				continue
			}

			// current token is logical
			if token.TType == efp.OperatorsInfix && token.TSubType == efp.TokenSubTypeLogical {
			}

			// current token is text
			if token.TType == efp.TokenTypeOperand && token.TSubType == efp.TokenSubTypeText {
				argsList.PushBack(token)
			}

			// current token is function stop
			if token.TType == efp.TokenTypeFunction && token.TSubType == efp.TokenSubTypeStop {
				for !opftStack.Empty() {
					// calculate trigger
					topOpt := opftStack.Peek().(efp.Token)
					if err := calculate(opfdStack, topOpt); err != nil {
						return efp.Token{}, err
					}
					opftStack.Pop()
				}

				// push opfd to args
				if opfdStack.Len() > 0 {
					argsList.PushBack(opfdStack.Pop())
				}
				// call formula function to evaluate
				result, err := callFuncByName(&formulaFuncs{}, strings.NewReplacer(
					"_xlfn", "", ".", "").Replace(opfStack.Peek().(efp.Token).TValue),
					[]reflect.Value{reflect.ValueOf(argsList)})
				if err != nil {
					return efp.Token{}, err
				}
				argsList.Init()
				opfStack.Pop()
				if opfStack.Len() > 0 { // still in function stack
					opfdStack.Push(efp.Token{TValue: result, TType: efp.TokenTypeOperand, TSubType: efp.TokenSubTypeNumber})
				} else {
					opdStack.Push(efp.Token{TValue: result, TType: efp.TokenTypeOperand, TSubType: efp.TokenSubTypeNumber})
				}
			}
		}
	}
	for optStack.Len() != 0 {
		topOpt := optStack.Peek().(efp.Token)
		if err = calculate(opdStack, topOpt); err != nil {
			return efp.Token{}, err
		}
		optStack.Pop()
	}
	return opdStack.Peek().(efp.Token), err
}

// calculate evaluate basic arithmetic operations.
func calculate(opdStack *Stack, opt efp.Token) error {
	if opt.TValue == "-" && opt.TType == efp.TokenTypeOperatorPrefix {
		opd := opdStack.Pop().(efp.Token)
		opdVal, err := strconv.ParseFloat(opd.TValue, 64)
		if err != nil {
			return err
		}
		result := 0 - opdVal
		opdStack.Push(efp.Token{TValue: fmt.Sprintf("%g", result), TType: efp.TokenTypeOperand, TSubType: efp.TokenSubTypeNumber})
	}
	if opt.TValue == "+" {
		rOpd := opdStack.Pop().(efp.Token)
		lOpd := opdStack.Pop().(efp.Token)
		lOpdVal, err := strconv.ParseFloat(lOpd.TValue, 64)
		if err != nil {
			return err
		}
		rOpdVal, err := strconv.ParseFloat(rOpd.TValue, 64)
		if err != nil {
			return err
		}
		result := lOpdVal + rOpdVal
		opdStack.Push(efp.Token{TValue: fmt.Sprintf("%g", result), TType: efp.TokenTypeOperand, TSubType: efp.TokenSubTypeNumber})
	}
	if opt.TValue == "-" && opt.TType == efp.TokenTypeOperatorInfix {
		rOpd := opdStack.Pop().(efp.Token)
		lOpd := opdStack.Pop().(efp.Token)
		lOpdVal, err := strconv.ParseFloat(lOpd.TValue, 64)
		if err != nil {
			return err
		}
		rOpdVal, err := strconv.ParseFloat(rOpd.TValue, 64)
		if err != nil {
			return err
		}
		result := lOpdVal - rOpdVal
		opdStack.Push(efp.Token{TValue: fmt.Sprintf("%g", result), TType: efp.TokenTypeOperand, TSubType: efp.TokenSubTypeNumber})
	}
	if opt.TValue == "*" {
		rOpd := opdStack.Pop().(efp.Token)
		lOpd := opdStack.Pop().(efp.Token)
		lOpdVal, err := strconv.ParseFloat(lOpd.TValue, 64)
		if err != nil {
			return err
		}
		rOpdVal, err := strconv.ParseFloat(rOpd.TValue, 64)
		if err != nil {
			return err
		}
		result := lOpdVal * rOpdVal
		opdStack.Push(efp.Token{TValue: fmt.Sprintf("%g", result), TType: efp.TokenTypeOperand, TSubType: efp.TokenSubTypeNumber})
	}
	if opt.TValue == "/" {
		rOpd := opdStack.Pop().(efp.Token)
		lOpd := opdStack.Pop().(efp.Token)
		lOpdVal, err := strconv.ParseFloat(lOpd.TValue, 64)
		if err != nil {
			return err
		}
		rOpdVal, err := strconv.ParseFloat(rOpd.TValue, 64)
		if err != nil {
			return err
		}
		result := lOpdVal / rOpdVal
		if rOpdVal == 0 {
			return errors.New(formulaErrorDIV)
		}
		opdStack.Push(efp.Token{TValue: fmt.Sprintf("%g", result), TType: efp.TokenTypeOperand, TSubType: efp.TokenSubTypeNumber})
	}
	return nil
}

// parseToken parse basic arithmetic operator priority and evaluate based on
// operators and operands.
func (f *File) parseToken(sheet string, token efp.Token, opdStack, optStack *Stack) error {
	// parse reference: must reference at here
	if token.TSubType == efp.TokenSubTypeRange {
		result, err := f.parseReference(sheet, token.TValue)
		if err != nil {
			return errors.New(formulaErrorNAME)
		}
		if len(result) != 1 {
			return errors.New(formulaErrorVALUE)
		}
		token.TValue = result[0]
		token.TType = efp.TokenTypeOperand
		token.TSubType = efp.TokenSubTypeNumber
	}
	if (token.TValue == "-" && token.TType == efp.TokenTypeOperatorPrefix) || token.TValue == "+" || token.TValue == "-" || token.TValue == "*" || token.TValue == "/" {
		if optStack.Len() == 0 {
			optStack.Push(token)
		} else {
			tokenPriority := getPriority(token)
			topOpt := optStack.Peek().(efp.Token)
			topOptPriority := getPriority(topOpt)
			if tokenPriority > topOptPriority {
				optStack.Push(token)
			} else {
				for tokenPriority <= topOptPriority {
					optStack.Pop()
					if err := calculate(opdStack, topOpt); err != nil {
						return err
					}
					if optStack.Len() > 0 {
						topOpt = optStack.Peek().(efp.Token)
						topOptPriority = getPriority(topOpt)
						continue
					}
					break
				}
				optStack.Push(token)
			}
		}
	}
	if token.TType == efp.TokenTypeSubexpression && token.TSubType == efp.TokenSubTypeStart { // (
		optStack.Push(token)
	}
	if token.TType == efp.TokenTypeSubexpression && token.TSubType == efp.TokenSubTypeStop { // )
		for optStack.Peek().(efp.Token).TSubType != efp.TokenSubTypeStart && optStack.Peek().(efp.Token).TType != efp.TokenTypeSubexpression { // != (
			topOpt := optStack.Peek().(efp.Token)
			if err := calculate(opdStack, topOpt); err != nil {
				return err
			}
			optStack.Pop()
		}
		optStack.Pop()
	}
	// opd
	if token.TType == efp.TokenTypeOperand && token.TSubType == efp.TokenSubTypeNumber {
		opdStack.Push(token)
	}
	return nil
}

// parseReference parse reference and extract values by given reference
// characters and default sheet name.
func (f *File) parseReference(sheet, reference string) (result []string, err error) {
	reference = strings.Replace(reference, "$", "", -1)
	refs, cellRanges, cellRefs := list.New(), list.New(), list.New()
	for _, ref := range strings.Split(reference, ":") {
		tokens := strings.Split(ref, "!")
		cr := cellRef{}
		if len(tokens) == 2 { // have a worksheet name
			cr.Sheet = tokens[0]
			if cr.Col, cr.Row, err = CellNameToCoordinates(tokens[1]); err != nil {
				return
			}
			if refs.Len() > 0 {
				e := refs.Back()
				cellRefs.PushBack(e.Value.(cellRef))
				refs.Remove(e)
			}
			refs.PushBack(cr)
			continue
		}
		if cr.Col, cr.Row, err = CellNameToCoordinates(tokens[0]); err != nil {
			return
		}
		e := refs.Back()
		if e == nil {
			cr.Sheet = sheet
			refs.PushBack(cr)
			continue
		}
		cellRanges.PushBack(cellRange{
			From: e.Value.(cellRef),
			To:   cr,
		})
		refs.Remove(e)
	}
	if refs.Len() > 0 {
		e := refs.Back()
		cellRefs.PushBack(e.Value.(cellRef))
		refs.Remove(e)
	}

	result, err = f.rangeResolver(cellRefs, cellRanges)
	return
}

// rangeResolver extract value as string from given reference and range list.
// This function will not ignore the empty cell. Note that the result of 3D
// range references may be different from Excel in some cases, for example,
// A1:A2:A2:B3 in Excel will include B1, but we wont.
func (f *File) rangeResolver(cellRefs, cellRanges *list.List) (result []string, err error) {
	filter := map[string]string{}
	// extract value from ranges
	for temp := cellRanges.Front(); temp != nil; temp = temp.Next() {
		cr := temp.Value.(cellRange)
		if cr.From.Sheet != cr.To.Sheet {
			err = errors.New(formulaErrorVALUE)
		}
		rng := []int{cr.From.Col, cr.From.Row, cr.To.Col, cr.To.Row}
		sortCoordinates(rng)
		for col := rng[0]; col <= rng[2]; col++ {
			for row := rng[1]; row <= rng[3]; row++ {
				var cell string
				if cell, err = CoordinatesToCellName(col, row); err != nil {
					return
				}
				if filter[cell], err = f.GetCellValue(cr.From.Sheet, cell); err != nil {
					return
				}
			}
		}
	}
	// extract value from references
	for temp := cellRefs.Front(); temp != nil; temp = temp.Next() {
		cr := temp.Value.(cellRef)
		var cell string
		if cell, err = CoordinatesToCellName(cr.Col, cr.Row); err != nil {
			return
		}
		if filter[cell], err = f.GetCellValue(cr.Sheet, cell); err != nil {
			return
		}
	}

	for _, val := range filter {
		result = append(result, val)
	}
	return
}

// callFuncByName calls the no error or only error return function with
// reflect by given receiver, name and parameters.
func callFuncByName(receiver interface{}, name string, params []reflect.Value) (result string, err error) {
	function := reflect.ValueOf(receiver).MethodByName(name)
	if function.IsValid() {
		rt := function.Call(params)
		if len(rt) == 0 {
			return
		}
		if !rt[1].IsNil() {
			err = rt[1].Interface().(error)
			return
		}
		result = rt[0].Interface().(string)
		return
	}
	err = fmt.Errorf("not support %s function", name)
	return
}

// Math and Trigonometric functions

// ABS function returns the absolute value of any supplied number. The syntax
// of the function is:
//
//   ABS(number)
//
func (fn *formulaFuncs) ABS(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ABS requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Abs(val))
	return
}

// ACOS function calculates the arccosine (i.e. the inverse cosine) of a given
// number, and returns an angle, in radians, between 0 and π. The syntax of
// the function is:
//
//   ACOS(number)
//
func (fn *formulaFuncs) ACOS(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ACOS requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Acos(val))
	return
}

// ACOSH function calculates the inverse hyperbolic cosine of a supplied number.
// of the function is:
//
//   ACOSH(number)
//
func (fn *formulaFuncs) ACOSH(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ACOSH requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Acosh(val))
	return
}

// ACOT function calculates the arccotangent (i.e. the inverse cotangent) of a
// given number, and returns an angle, in radians, between 0 and π. The syntax
// of the function is:
//
//   ACOT(number)
//
func (fn *formulaFuncs) ACOT(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ACOT requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Pi/2-math.Atan(val))
	return
}

// ACOTH function calculates the hyperbolic arccotangent (coth) of a supplied
// value. The syntax of the function is:
//
//   ACOTH(number)
//
func (fn *formulaFuncs) ACOTH(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ACOTH requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Atanh(1/val))
	return
}

// ARABIC function converts a Roman numeral into an Arabic numeral. The syntax
// of the function is:
//
//   ARABIC(text)
//
func (fn *formulaFuncs) ARABIC(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ARABIC requires 1 numeric arguments")
		return
	}
	val, last, prefix := 0.0, 0.0, 1.0
	for _, char := range argsList.Front().Value.(efp.Token).TValue {
		digit := 0.0
		switch char {
		case '-':
			prefix = -1
			continue
		case 'I':
			digit = 1
		case 'V':
			digit = 5
		case 'X':
			digit = 10
		case 'L':
			digit = 50
		case 'C':
			digit = 100
		case 'D':
			digit = 500
		case 'M':
			digit = 1000
		}
		val += digit
		switch {
		case last == digit && (last == 5 || last == 50 || last == 500):
			result = formulaErrorVALUE
			return
		case 2*last == digit:
			result = formulaErrorVALUE
			return
		}
		if last < digit {
			val -= 2 * last
		}
		last = digit
	}
	result = fmt.Sprintf("%g", prefix*val)
	return
}

// ASIN function calculates the arcsine (i.e. the inverse sine) of a given
// number, and returns an angle, in radians, between -π/2 and π/2. The syntax
// of the function is:
//
//   ASIN(number)
//
func (fn *formulaFuncs) ASIN(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ASIN requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Asin(val))
	return
}

// ASINH function calculates the inverse hyperbolic sine of a supplied number.
// The syntax of the function is:
//
//   ASINH(number)
//
func (fn *formulaFuncs) ASINH(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ASINH requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Asinh(val))
	return
}

// ATAN function calculates the arctangent (i.e. the inverse tangent) of a
// given number, and returns an angle, in radians, between -π/2 and +π/2. The
// syntax of the function is:
//
//   ATAN(number)
//
func (fn *formulaFuncs) ATAN(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ATAN requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Atan(val))
	return
}

// ATANH function calculates the inverse hyperbolic tangent of a supplied
// number. The syntax of the function is:
//
//   ATANH(number)
//
func (fn *formulaFuncs) ATANH(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("ATANH requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Atanh(val))
	return
}

// ATAN2 function calculates the arctangent (i.e. the inverse tangent) of a
// given set of x and y coordinates, and returns an angle, in radians, between
// -π/2 and +π/2. The syntax of the function is:
//
//   ATAN2(x_num,y_num)
//
func (fn *formulaFuncs) ATAN2(argsList *list.List) (result string, err error) {
	if argsList.Len() != 2 {
		err = errors.New("ATAN2 requires 2 numeric arguments")
		return
	}
	var x, y float64
	x, err = strconv.ParseFloat(argsList.Back().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	y, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	result = fmt.Sprintf("%g", math.Atan2(x, y))
	return
}

// gcd returns the greatest common divisor of two supplied integers.
func gcd(x, y float64) float64 {
	x, y = math.Trunc(x), math.Trunc(y)
	if x == 0 {
		return y
	}
	if y == 0 {
		return x
	}
	for x != y {
		if x > y {
			x = x - y
		} else {
			y = y - x
		}
	}
	return x
}

// BASE function converts a number into a supplied base (radix), and returns a
// text representation of the calculated value. The syntax of the function is:
//
//   BASE(number,radix,[min_length])
//
func (fn *formulaFuncs) BASE(argsList *list.List) (result string, err error) {
	if argsList.Len() < 2 {
		err = errors.New("BASE requires at least 2 arguments")
		return
	}
	if argsList.Len() > 3 {
		err = errors.New("BASE allows at most 3 arguments")
		return
	}
	var number float64
	var radix, minLength int
	number, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	radix, err = strconv.Atoi(argsList.Front().Next().Value.(efp.Token).TValue)
	if err != nil {
		return
	}
	if radix < 2 || radix > 36 {
		err = errors.New("radix must be an integer ≥ 2 and ≤ 36")
		return
	}
	if argsList.Len() > 2 {
		minLength, err = strconv.Atoi(argsList.Back().Value.(efp.Token).TValue)
		if err != nil {
			return
		}
	}
	result = strconv.FormatInt(int64(number), radix)
	if len(result) < minLength {
		result = strings.Repeat("0", minLength-len(result)) + result
	}
	result = strings.ToUpper(result)
	return
}

// CEILING function rounds a supplied number away from zero, to the nearest
// multiple of a given number. The syntax of the function is:
//
//   CEILING(number,significance)
//
func (fn *formulaFuncs) CEILING(argsList *list.List) (result string, err error) {
	if argsList.Len() == 0 {
		err = errors.New("CEILING requires at least 1 argument")
		return
	}
	if argsList.Len() > 2 {
		err = errors.New("CEILING allows at most 2 arguments")
		return
	}
	var number, significance float64
	number, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	significance = 1
	if number < 0 {
		significance = -1
	}
	if argsList.Len() > 1 {
		significance, err = strconv.ParseFloat(argsList.Back().Value.(efp.Token).TValue, 64)
		if err != nil {
			return
		}
	}
	if significance < 0 && number > 0 {
		err = errors.New("negative sig to CEILING invalid")
		return
	}
	if argsList.Len() == 1 {
		result = fmt.Sprintf("%g", math.Ceil(number))
		return
	}
	number, res := math.Modf(number / significance)
	if res > 0 {
		number++
	}
	result = fmt.Sprintf("%g", number*significance)
	return
}

// CEILINGMATH function rounds a supplied number up to a supplied multiple of
// significance. The syntax of the function is:
//
//   CEILING.MATH(number,[significance],[mode])
//
func (fn *formulaFuncs) CEILINGMATH(argsList *list.List) (result string, err error) {
	if argsList.Len() == 0 {
		err = errors.New("CEILING.MATH requires at least 1 argument")
		return
	}
	if argsList.Len() > 3 {
		err = errors.New("CEILING.MATH allows at most 3 arguments")
		return
	}
	var number, significance, mode float64 = 0, 1, 1
	number, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	if number < 0 {
		significance = -1
	}
	if argsList.Len() > 1 {
		significance, err = strconv.ParseFloat(argsList.Front().Next().Value.(efp.Token).TValue, 64)
		if err != nil {
			return
		}
	}
	if argsList.Len() == 1 {
		result = fmt.Sprintf("%g", math.Ceil(number))
		return
	}
	if argsList.Len() > 2 {
		mode, err = strconv.ParseFloat(argsList.Back().Value.(efp.Token).TValue, 64)
		if err != nil {
			return
		}
	}
	val, res := math.Modf(number / significance)
	_, _ = res, mode
	if res != 0 {
		if number > 0 {
			val++
		} else if mode < 0 {
			val--
		}
	}

	result = fmt.Sprintf("%g", val*significance)
	return
}

// GCD function returns the greatest common divisor of two or more supplied
// integers. The syntax of the function is:
//
//   GCD(number1,[number2],...)
//
func (fn *formulaFuncs) GCD(argsList *list.List) (result string, err error) {
	if argsList.Len() == 0 {
		err = errors.New("GCD requires at least 1 argument")
		return
	}
	var (
		val  float64
		nums = []float64{}
	)
	for arg := argsList.Front(); arg != nil; arg = arg.Next() {
		token := arg.Value.(efp.Token)
		if token.TValue == "" {
			continue
		}
		val, err = strconv.ParseFloat(token.TValue, 64)
		if err != nil {
			return
		}
		nums = append(nums, val)
	}
	if nums[0] < 0 {
		err = errors.New("GCD only accepts positive arguments")
		return
	}
	if len(nums) == 1 {
		result = fmt.Sprintf("%g", nums[0])
		return
	}
	cd := nums[0]
	for i := 1; i < len(nums); i++ {
		if nums[i] < 0 {
			err = errors.New("GCD only accepts positive arguments")
			return
		}
		cd = gcd(cd, nums[i])
	}
	result = fmt.Sprintf("%g", cd)
	return
}

// lcm returns the least common multiple of two supplied integers.
func lcm(a, b float64) float64 {
	a = math.Trunc(a)
	b = math.Trunc(b)
	if a == 0 && b == 0 {
		return 0
	}
	return a * b / gcd(a, b)
}

// LCM function returns the least common multiple of two or more supplied
// integers. The syntax of the function is:
//
//   LCM(number1,[number2],...)
//
func (fn *formulaFuncs) LCM(argsList *list.List) (result string, err error) {
	if argsList.Len() == 0 {
		err = errors.New("LCM requires at least 1 argument")
		return
	}
	var (
		val  float64
		nums = []float64{}
	)
	for arg := argsList.Front(); arg != nil; arg = arg.Next() {
		token := arg.Value.(efp.Token)
		if token.TValue == "" {
			continue
		}
		val, err = strconv.ParseFloat(token.TValue, 64)
		if err != nil {
			return
		}
		nums = append(nums, val)
	}
	if nums[0] < 0 {
		err = errors.New("LCM only accepts positive arguments")
		return
	}
	if len(nums) == 1 {
		result = fmt.Sprintf("%g", nums[0])
		return
	}
	cm := nums[0]
	for i := 1; i < len(nums); i++ {
		if nums[i] < 0 {
			err = errors.New("LCM only accepts positive arguments")
			return
		}
		cm = lcm(cm, nums[i])
	}
	result = fmt.Sprintf("%g", cm)
	return
}

// POWER function calculates a given number, raised to a supplied power.
// The syntax of the function is:
//
//    POWER(number,power)
//
func (fn *formulaFuncs) POWER(argsList *list.List) (result string, err error) {
	if argsList.Len() != 2 {
		err = errors.New("POWER requires 2 numeric arguments")
		return
	}
	var x, y float64
	x, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	y, err = strconv.ParseFloat(argsList.Back().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	if x == 0 && y == 0 {
		err = errors.New(formulaErrorNUM)
		return
	}
	if x == 0 && y < 0 {
		err = errors.New(formulaErrorDIV)
		return
	}
	result = fmt.Sprintf("%g", math.Pow(x, y))
	return
}

// PRODUCT function returns the product (multiplication) of a supplied set of
// numerical values. The syntax of the function is:
//
//    PRODUCT(number1,[number2],...)
//
func (fn *formulaFuncs) PRODUCT(argsList *list.List) (result string, err error) {
	var (
		val     float64
		product float64 = 1
	)
	for arg := argsList.Front(); arg != nil; arg = arg.Next() {
		token := arg.Value.(efp.Token)
		if token.TValue == "" {
			continue
		}
		val, err = strconv.ParseFloat(token.TValue, 64)
		if err != nil {
			return
		}
		product = product * val
	}
	result = fmt.Sprintf("%g", product)
	return
}

// SIGN function returns the arithmetic sign (+1, -1 or 0) of a supplied
// number. I.e. if the number is positive, the Sign function returns +1, if
// the number is negative, the function returns -1 and if the number is 0
// (zero), the function returns 0. The syntax of the function is:
//
//   SIGN(number)
//
func (fn *formulaFuncs) SIGN(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("SIGN requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	if val < 0 {
		result = "-1"
		return
	}
	if val > 0 {
		result = "1"
		return
	}
	result = "0"
	return
}

// SQRT function calculates the positive square root of a supplied number. The
// syntax of the function is:
//
//    SQRT(number)
//
func (fn *formulaFuncs) SQRT(argsList *list.List) (result string, err error) {
	if argsList.Len() != 1 {
		err = errors.New("SQRT requires 1 numeric arguments")
		return
	}
	var val float64
	val, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	if val < 0 {
		err = errors.New(formulaErrorNUM)
		return
	}
	result = fmt.Sprintf("%g", math.Sqrt(val))
	return
}

// SUM function adds together a supplied set of numbers and returns the sum of
// these values. The syntax of the function is:
//
//    SUM(number1,[number2],...)
//
func (fn *formulaFuncs) SUM(argsList *list.List) (result string, err error) {
	var val float64
	var sum float64
	for arg := argsList.Front(); arg != nil; arg = arg.Next() {
		token := arg.Value.(efp.Token)
		if token.TValue == "" {
			continue
		}
		val, err = strconv.ParseFloat(token.TValue, 64)
		if err != nil {
			return
		}
		sum += val
	}
	result = fmt.Sprintf("%g", sum)
	return
}

// QUOTIENT function returns the integer portion of a division between two
// supplied numbers. The syntax of the function is:
//
//   QUOTIENT(numerator,denominator)
//
func (fn *formulaFuncs) QUOTIENT(argsList *list.List) (result string, err error) {
	if argsList.Len() != 2 {
		err = errors.New("QUOTIENT requires 2 numeric arguments")
		return
	}
	var x, y float64
	x, err = strconv.ParseFloat(argsList.Front().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	y, err = strconv.ParseFloat(argsList.Back().Value.(efp.Token).TValue, 64)
	if err != nil {
		return
	}
	if y == 0 {
		err = errors.New(formulaErrorDIV)
		return
	}
	result = fmt.Sprintf("%g", math.Trunc(x/y))
	return
}
