// Copyright (c) 2020, Peter Ohler, All rights reserved.

package tt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ohler55/ojg/gd"
)

func Equal(t *testing.T, expect, actual interface{}, args ...interface{}) (eq bool) {
	switch te := expect.(type) {
	case nil:
		eq = nil == actual
	case bool:
		switch ta := actual.(type) {
		case bool:
			eq = te == ta
		case gd.Bool:
			eq = te == bool(ta)
		default:
			eq = false
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, gd.Int:
		x, _ := asInt(expect)
		a, ok := asInt(actual)
		eq = x == a && ok
	case float32, float64:
		x, _ := asFloat(expect)
		a, ok := asFloat(actual)
		eq = x == a && ok
	case string:
		x, _ := asString(expect)
		a, ok := asString(actual)
		eq = x == a && ok
		if !eq {
			/*
				if !eq {
					tx, ta = colorizeStrings(tx, ta)
					expect = tx
					actual = ta
				}
			*/
		}
	case []interface{}:
		switch ta := actual.(type) {
		case []interface{}:
			eq = true
			for i := 0; i < len(te); i++ {
				if len(ta) <= i {
					eq = false
					break
				}
				if eq = Equal(t, te[i], ta[i], args...); !eq {
					break
				}
			}
			if eq && len(te) != len(ta) {
				eq = false
			}
		case gd.Array:
			eq = Equal(t, expect, ta.Simplify(), args...)
		default:
			eq = false
		}
	case map[string]interface{}:
		switch ta := actual.(type) {
		case map[string]interface{}:
			eq = true
			for k, ve := range te {
				va, has := ta[k]
				if !has {
					eq = false
					break
				}
				eq = Equal(t, ve, va, args...)
			}
			if eq && len(te) != len(ta) {
				eq = false
			}
		case gd.Object:
			eq = Equal(t, expect, ta.Simplify(), args...)
		default:
			eq = false
		}
	default:
		// TBD maps
	}
	if !eq {
		var b strings.Builder
		b.WriteString(fmt.Sprintf("\nexpect: (%T) %v\nactual: (%T) %v\n", expect, expect, actual, actual))
		stackFill(&b)
		if 0 < len(args) {
			if format, _ := args[0].(string); 0 < len(format) {
				b.WriteString(fmt.Sprintf(format, args[1:]...))
			} else {
				b.WriteString(fmt.Sprint(args...))
			}
		}
		t.Fatal(b.String())
	}
	return
}