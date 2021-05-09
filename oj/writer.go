// Copyright (c) 2020, Peter Ohler, All rights reserved.

package oj

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/alt"
)

const (
	spaces = "\n                                                                                                                                "
	tabs   = "\n\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t"
)

// Writer is a JSON writer that includes a reused buffer for reduced
// allocations for repeated encoding calls.
type Writer struct {
	ojg.Options
	buf           []byte
	w             io.Writer
	findex        byte
	strict        bool
	appendArray   func(wr *Writer, data []interface{}, depth int)
	appendObject  func(wr *Writer, data map[string]interface{}, depth int)
	appendDefault func(wr *Writer, data interface{}, depth int)
}

// JSON writes data, JSON encoded. On error, an empty string is returned.
func (wr *Writer) JSON(data interface{}) string {
	defer func() {
		if r := recover(); r != nil {
			wr.buf = wr.buf[:0]
		}
	}()
	return string(wr.MustJSON(data))
}

// MustJSON writes data, JSON encoded as a []byte and not a string like the
// JSON() function. On error a panic is called with the error.
func (wr *Writer) MustJSON(data interface{}) []byte {
	wr.w = nil
	if wr.InitSize <= 0 {
		wr.InitSize = 256
	}
	if cap(wr.buf) < wr.InitSize {
		wr.buf = make([]byte, 0, wr.InitSize)
	} else {
		wr.buf = wr.buf[:0]
	}
	if wr.findex == 0 {
		wr.findex = wr.FieldsIndex()
	}
	if wr.Tab || 0 < wr.Indent {
		wr.appendArray = appendArray
		if wr.Sort {
			wr.appendObject = appendSortObject
		} else {
			wr.appendObject = appendObject
		}
		wr.appendDefault = appendDefault
	} else {
		wr.appendArray = tightArray
		if wr.Sort {
			wr.appendObject = tightSortObject
		} else {
			wr.appendObject = tightObject
		}
		wr.appendDefault = tightDefault
	}
	wr.appendJSON(data, 0)

	return wr.buf
}

// Write a JSON string for the data provided.
func (wr *Writer) Write(w io.Writer, data interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			wr.buf = wr.buf[:0]
			if err, _ = r.(error); err == nil {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	wr.MustWrite(w, data)
	return
}

// MustWrite a JSON string for the data provided. If an error occurs panic is
// called with the error.
func (wr *Writer) MustWrite(w io.Writer, data interface{}) {
	wr.w = w
	if wr.InitSize <= 0 {
		wr.InitSize = 256
	}
	if wr.WriteLimit <= 0 {
		wr.WriteLimit = 1024
	}
	if cap(wr.buf) < wr.InitSize {
		wr.buf = make([]byte, 0, wr.InitSize)
	} else {
		wr.buf = wr.buf[:0]
	}
	if wr.findex == 0 {
		wr.findex = wr.FieldsIndex()
	}
	if wr.Color {
		wr.colorJSON(data, 0)
	} else {
		if wr.Tab || 0 < wr.Indent {
			wr.appendArray = appendArray
			wr.appendObject = appendObject
			wr.appendDefault = appendDefault
		} else {
			wr.appendArray = tightArray
			wr.appendObject = tightObject
			wr.appendDefault = tightDefault
		}
		wr.appendJSON(data, 0)
	}
	if 0 < len(wr.buf) {
		if _, err := wr.w.Write(wr.buf); err != nil {
			panic(err)
		}
	}
}

func (wr *Writer) appendJSON(data interface{}, depth int) {

	// TBD if marshal and nil (as apposed to empty) the null
	//  use wr.strict field as indicator of marshal called?

	switch td := data.(type) {
	case nil:
		wr.buf = append(wr.buf, []byte("null")...)

	case bool:
		if td {
			wr.buf = append(wr.buf, []byte("true")...)
		} else {
			wr.buf = append(wr.buf, []byte("false")...)
		}

	case int:
		wr.buf = strconv.AppendInt(wr.buf, int64(td), 10)
	case int8:
		wr.buf = strconv.AppendInt(wr.buf, int64(td), 10)
	case int16:
		wr.buf = strconv.AppendInt(wr.buf, int64(td), 10)
	case int32:
		wr.buf = strconv.AppendInt(wr.buf, int64(td), 10)
	case int64:
		wr.buf = strconv.AppendInt(wr.buf, td, 10)
	case uint:
		wr.buf = strconv.AppendUint(wr.buf, uint64(td), 10)
	case uint8:
		wr.buf = strconv.AppendUint(wr.buf, uint64(td), 10)
	case uint16:
		wr.buf = strconv.AppendUint(wr.buf, uint64(td), 10)
	case uint32:
		wr.buf = strconv.AppendUint(wr.buf, uint64(td), 10)
	case uint64:
		wr.buf = strconv.AppendUint(wr.buf, td, 10)

	case float32:
		wr.buf = strconv.AppendFloat(wr.buf, float64(td), 'g', -1, 32)
	case float64:
		wr.buf = strconv.AppendFloat(wr.buf, float64(td), 'g', -1, 64)

	case string:
		wr.buf = ojg.AppendJSONString(wr.buf, td, !wr.HTMLUnsafe)

	case time.Time:
		wr.appendTime(td)

	case []interface{}:
		wr.appendArray(wr, td, depth)

	case map[string]interface{}:
		wr.appendObject(wr, td, depth)

	default:
		wr.appendDefault(wr, data, depth)
	}
	if wr.w != nil && wr.WriteLimit < len(wr.buf) {
		if _, err := wr.w.Write(wr.buf); err != nil {
			panic(err)
		}
		wr.buf = wr.buf[:0]
	}
}

func appendDefault(wr *Writer, data interface{}, depth int) {
	if g, _ := data.(alt.Genericer); g != nil {
		wr.appendJSON(g.Generic().Simplify(), depth)
		return
	}
	if simp, _ := data.(alt.Simplifier); simp != nil {
		data = simp.Simplify()
		wr.appendJSON(data, depth)
		return
	}
	if !wr.NoReflect {
		rv := reflect.ValueOf(data)
		kind := rv.Kind()
		if kind == reflect.Ptr {
			rv = rv.Elem()
			kind = rv.Kind()
		}
		switch kind {
		case reflect.Struct:
			wr.appendStruct(rv, depth, nil)
		case reflect.Slice, reflect.Array:
			wr.appendSlice(rv, depth, nil)
		case reflect.Map:
			wr.appendMap(rv, depth, nil)
		default:
			// Not much should get here except Complex and non-decomposable
			// values.
			dec := alt.Decompose(data, &wr.Options)
			wr.appendJSON(dec, depth)
			return
		}
	} else if wr.strict {
		panic(fmt.Errorf("%T can not be encoded as a JSON element", data))
	} else {
		wr.buf = ojg.AppendJSONString(wr.buf, fmt.Sprintf("%v", data), !wr.HTMLUnsafe)
	}
}

func (wr *Writer) appendTime(t time.Time) {
	if wr.TimeMap {
		wr.buf = append(wr.buf, []byte(`{"`)...)
		wr.buf = append(wr.buf, wr.CreateKey...)
		wr.buf = append(wr.buf, []byte(`":"`)...)
		if wr.FullTypePath {
			wr.buf = append(wr.buf, []byte("time/Time")...)
		} else {
			wr.buf = append(wr.buf, []byte("Time")...)
		}
		wr.buf = append(wr.buf, []byte(`","value":`)...)
	} else if 0 < len(wr.TimeWrap) {
		wr.buf = append(wr.buf, []byte(`{"`)...)
		wr.buf = append(wr.buf, []byte(wr.TimeWrap)...)
		wr.buf = append(wr.buf, []byte(`":`)...)
	}
	switch wr.TimeFormat {
	case "", "nano":
		wr.buf = append(wr.buf, []byte(strconv.FormatInt(t.UnixNano(), 10))...)
	case "second":
		// Decimal format but float is not accurate enough so append the output
		// in two parts.
		nano := t.UnixNano()
		secs := nano / int64(time.Second)
		if 0 < nano {
			wr.buf = append(wr.buf, []byte(fmt.Sprintf("%d.%09d", secs, nano-(secs*int64(time.Second))))...)
		} else {
			wr.buf = append(wr.buf, []byte(fmt.Sprintf("%d.%09d", secs, -(nano-(secs*int64(time.Second)))))...)
		}
	default:
		wr.buf = append(wr.buf, '"')
		wr.buf = append(wr.buf, []byte(t.Format(wr.TimeFormat))...)
		wr.buf = append(wr.buf, '"')
	}
	if 0 < len(wr.TimeWrap) || wr.TimeMap {
		wr.buf = append(wr.buf, '}')
	}
}

func appendArray(wr *Writer, n []interface{}, depth int) {
	var is string
	var cs string
	d2 := depth + 1
	if wr.Tab {
		x := depth + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		is = tabs[1:x]
		x = d2 + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		cs = tabs[0:x]
	} else {
		x := depth*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		is = spaces[1:x]
		x = d2*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		cs = spaces[0:x]
	}
	if 0 < len(n) {
		wr.buf = append(wr.buf, '[')
		for _, m := range n {
			wr.buf = append(wr.buf, cs...)
			wr.appendJSON(m, d2)
			wr.buf = append(wr.buf, ',')
		}
		wr.buf[len(wr.buf)-1] = '\n'
		wr.buf = append(wr.buf, is...)
		wr.buf = append(wr.buf, ']')
	} else {
		wr.buf = append(wr.buf, "[]"...)
	}
}

func appendObject(wr *Writer, n map[string]interface{}, depth int) {
	d2 := depth + 1
	var is string
	var cs string
	if wr.Tab {
		x := depth + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		is = tabs[1:x]
		x = d2 + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		cs = tabs[0:x]
	} else {
		x := depth*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		is = spaces[1:x]
		x = d2*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		cs = spaces[0:x]
	}
	empty := true
	wr.buf = append(wr.buf, '{')
	for k, m := range n {
		if m == nil && wr.OmitNil {
			continue
		}
		empty = false
		wr.buf = append(wr.buf, []byte(cs)...)
		wr.buf = ojg.AppendJSONString(wr.buf, k, !wr.HTMLUnsafe)
		wr.buf = append(wr.buf, ':')
		wr.buf = append(wr.buf, ' ')
		wr.appendJSON(m, d2)
		wr.buf = append(wr.buf, ',')
	}
	if !empty {
		wr.buf[len(wr.buf)-1] = '\n'
		wr.buf = append(wr.buf, []byte(is)...)
	}
	wr.buf = append(wr.buf, '}')
}

func appendSortObject(wr *Writer, n map[string]interface{}, depth int) {
	d2 := depth + 1
	var is string
	var cs string
	if wr.Tab {
		x := depth + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		is = tabs[1:x]
		x = d2 + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		cs = tabs[0:x]
	} else {
		x := depth*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		is = spaces[1:x]
		x = d2*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		cs = spaces[0:x]
	}
	keys := make([]string, 0, len(n))
	for k := range n {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	empty := true
	wr.buf = append(wr.buf, '{')
	for _, k := range keys {
		m := n[k]
		if m == nil && wr.OmitNil {
			continue
		}
		empty = false
		wr.buf = append(wr.buf, []byte(cs)...)
		wr.buf = ojg.AppendJSONString(wr.buf, k, !wr.HTMLUnsafe)
		wr.buf = append(wr.buf, ':')
		wr.buf = append(wr.buf, ' ')
		wr.appendJSON(m, d2)
		wr.buf = append(wr.buf, ',')
	}
	if !empty {
		wr.buf[len(wr.buf)-1] = '\n'
		wr.buf = append(wr.buf, []byte(is)...)
	}
	wr.buf = append(wr.buf, '}')
}

func (wr *Writer) appendStruct(rv reflect.Value, depth int, st *ojg.Struct) {
	if st == nil {
		st = ojg.GetStruct(rv.Interface())
	}
	d2 := depth + 1
	fields := st.Fields[wr.findex&ojg.MaskIndex]
	wr.buf = append(wr.buf, '{')
	empty := true
	var v interface{}
	var has bool
	var wrote bool
	indented := false
	var is string
	var cs string
	if wr.Tab {
		x := depth + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		is = tabs[1:x]
		x = d2 + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		cs = tabs[0:x]
	} else {
		x := depth*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		is = spaces[1:x]
		x = d2*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		cs = spaces[0:x]
	}
	if 0 < len(wr.CreateKey) {
		wr.buf = append(wr.buf, []byte(cs)...)
		wr.buf = append(wr.buf, '"')
		wr.buf = append(wr.buf, wr.CreateKey...)
		wr.buf = append(wr.buf, `": "`...)
		if wr.FullTypePath {
			wr.buf = append(wr.buf, (st.Type.PkgPath() + "/" + st.Type.Name())...)
		} else {
			wr.buf = append(wr.buf, st.Type.Name()...)
		}
		wr.buf = append(wr.buf, `",`...)
		empty = false
	}
	for _, fi := range fields {
		if !indented {
			wr.buf = append(wr.buf, []byte(cs)...)
			indented = true
		}
		wr.buf, v, wrote, has = fi.Append(fi, wr.buf, rv, !wr.HTMLUnsafe)
		if wrote {
			wr.buf = append(wr.buf, ',')
			empty = false
			indented = false
			continue
		}
		if !has {
			continue
		}
		indented = false
		var fv reflect.Value
		kind := fi.Kind
		if kind == reflect.Ptr {
			fv = reflect.ValueOf(v).Elem()
			kind = fv.Kind()
			v = fv.Interface()
		}
		switch kind {
		case reflect.Struct:
			if !fv.IsValid() {
				fv = reflect.ValueOf(v)
			}
			wr.appendStruct(fv, d2, fi.Elem)
		case reflect.Slice, reflect.Array:
			if !fv.IsValid() {
				fv = reflect.ValueOf(v)
			}
			wr.appendSlice(fv, d2, fi.Elem)
		case reflect.Map:
			if !fv.IsValid() {
				fv = reflect.ValueOf(v)
			}
			wr.appendMap(fv, d2, fi.Elem)
		default:
			wr.appendJSON(v, d2)
		}
		wr.buf = append(wr.buf, ',')
		empty = false
	}
	if indented {
		wr.buf = wr.buf[:len(wr.buf)-len(cs)]
	}
	if !empty {
		wr.buf[len(wr.buf)-1] = '\n'
		wr.buf = append(wr.buf, []byte(is)...)
	}
	wr.buf = append(wr.buf, '}')
}

func (wr *Writer) appendSlice(rv reflect.Value, depth int, st *ojg.Struct) {
	d2 := depth + 1
	end := rv.Len()
	var is string
	var cs string
	if wr.Tab {
		x := depth + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		is = tabs[1:x]
		x = d2 + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		cs = tabs[0:x]
	} else {
		x := depth*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		is = spaces[1:x]
		x = d2*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		cs = spaces[0:x]
	}
	empty := true
	wr.buf = append(wr.buf, '[')
	for j := 0; j < end; j++ {
		wr.buf = append(wr.buf, []byte(cs)...)
		rm := rv.Index(j)
		switch rm.Kind() {
		case reflect.Struct:
			wr.appendStruct(rm, d2, st)
		case reflect.Slice, reflect.Array:
			wr.appendSlice(rm, d2, st)
		case reflect.Map:
			wr.appendMap(rm, d2, st)
		default:
			wr.appendJSON(rm.Interface(), d2)
		}
		wr.buf = append(wr.buf, ',')
		empty = false
	}
	if !empty {
		wr.buf[len(wr.buf)-1] = '\n'
		wr.buf = append(wr.buf, []byte(is)...)
	}
	wr.buf = append(wr.buf, ']')
}

func (wr *Writer) appendMap(rv reflect.Value, depth int, st *ojg.Struct) {
	d2 := depth + 1
	var is string
	var cs string
	if wr.Tab {
		x := depth + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		is = tabs[1:x]
		x = d2 + 1
		if len(tabs) < x {
			x = len(tabs)
		}
		cs = tabs[0:x]
	} else {
		x := depth*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		is = spaces[1:x]
		x = d2*wr.Indent + 1
		if len(spaces) < x {
			x = len(spaces)
		}
		cs = spaces[0:x]
	}
	keys := rv.MapKeys()
	if wr.Sort {
		sort.Slice(keys, func(i, j int) bool { return 0 < strings.Compare(keys[i].String(), keys[j].String()) })
	}
	empty := true
	wr.buf = append(wr.buf, '{')
	for _, kv := range keys {
		rm := rv.MapIndex(kv)
		if rm.Kind() == reflect.Ptr {
			if wr.OmitNil && rm.IsNil() {
				continue
			}
			rm = rm.Elem()
		}
		wr.buf = append(wr.buf, []byte(cs)...)
		switch rm.Kind() {
		case reflect.Struct:
			wr.buf = ojg.AppendJSONString(wr.buf, kv.String(), !wr.HTMLUnsafe)
			wr.buf = append(wr.buf, ": "...)
			wr.appendStruct(rm, d2, st)
		case reflect.Slice, reflect.Array:
			if wr.OmitNil && rm.IsNil() {
				continue
			}
			wr.buf = ojg.AppendJSONString(wr.buf, kv.String(), !wr.HTMLUnsafe)
			wr.buf = append(wr.buf, ": "...)
			wr.appendSlice(rm, d2, st)
		case reflect.Map:
			if wr.OmitNil && rm.IsNil() {
				continue
			}
			wr.buf = ojg.AppendJSONString(wr.buf, kv.String(), !wr.HTMLUnsafe)
			wr.buf = append(wr.buf, ": "...)
			wr.appendMap(rm, d2, st)
		default:
			wr.buf = ojg.AppendJSONString(wr.buf, kv.String(), !wr.HTMLUnsafe)
			wr.buf = append(wr.buf, ": "...)
			wr.appendJSON(rm.Interface(), d2)
		}
		wr.buf = append(wr.buf, ',')
		empty = false
	}
	if !empty {
		wr.buf[len(wr.buf)-1] = '\n'
		wr.buf = append(wr.buf, []byte(is)...)
	}
	wr.buf = append(wr.buf, '}')
}
