// Copyright (c) 2021, Peter Ohler, All rights reserved.

package alt

import (
	"reflect"

	"github.com/ohler55/ojg"
)

// Options is an alias for ojg.Options
type Options = ojg.Options

// Converter is an alias for ojg.Converter
type Converter = ojg.Converter

var (
	// DefaultOptions are the default options for the this package.
	DefaultOptions = ojg.DefaultOptions
	// BrightOptions are the bright color options.
	BrightOptions = ojg.BrightOptions
	// GoOptions are the options that match the go json.Marshal behavior.
	GoOptions = ojg.GoOptions
	// HTMLOptions are the options that can be used to encode as HTML JSON.
	HTMLOptions = ojg.HTMLOptions

	// TimeRFC3339Converter converts RFC3339 string into time.Time when
	// parsing.
	TimeRFC3339Converter = ojg.TimeRFC3339Converter
	// TimeNanoConverter converts integer values to time.Time assuming the
	// integer are nonoseconds,
	TimeNanoConverter = ojg.TimeNanoConverter
	// MongoConverter converts mongodb decorations into the correct times.
	MongoConverter = ojg.MongoConverter
)

func init() {
	// Use different defaults for decompose except the Go defaults. Set
	// OmitNil and provide a CreateKey for all.
	DefaultOptions.OmitNil = true
	DefaultOptions.CreateKey = "type"
	BrightOptions.OmitNil = true
	BrightOptions.CreateKey = "type"
	HTMLOptions.OmitNil = true
	HTMLOptions.CreateKey = "type"
}

// Dup is an alias for Decompose.
func Dup(v interface{}, options ...*ojg.Options) interface{} {
	return Decompose(v, options...)
}

// Decompose creates a simple type converting non simple to simple types using
// either the Simplify() interface or reflection. Unlike Alter() a deep copy
// is returned leaving the original data unchanged.
func Decompose(v interface{}, options ...*ojg.Options) interface{} {
	opt := &DefaultOptions
	if 0 < len(options) {
		opt = options[0]
	}
	if opt.Converter != nil {
		v = opt.Converter.Convert(v)
	}
	return decompose(v, opt)
}

// Alter the data into all simple types converting non simple to simple types
// using either the Simplify() interface or reflection. Unlike Decompose() map and
// slices members are modified if necessary to assure all elements are simple
// types.
func Alter(v interface{}, options ...*ojg.Options) interface{} {
	opt := &DefaultOptions
	if 0 < len(options) {
		opt = options[0]
	}
	if opt.Converter != nil {
		v = opt.Converter.Convert(v)
	}
	return alter(v, opt)
}

// Recompose simple data into more complex go types.
func Recompose(v interface{}, tv ...interface{}) (out interface{}, err error) {
	return DefaultRecomposer.Recompose(v, tv...)
}

// MustRecompose simple data into more complex go types and panics on error.
func MustRecompose(v interface{}, tv ...interface{}) (out interface{}) {
	return DefaultRecomposer.MustRecompose(v, tv...)
}

// NewRecomposer creates a new instance. The composers are a map of objects
// expected and functions to recompose them. If no function is provided then
// reflection is used instead.
func NewRecomposer(
	createKey string,
	composers map[interface{}]RecomposeFunc,
	anyComposers ...map[interface{}]RecomposeAnyFunc) (rec *Recomposer, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ojg.NewError(r)
		}
	}()
	rec = MustNewRecomposer(createKey, composers, anyComposers...)

	return
}

// MustNewRecomposer creates a new instance. The composers are a map of objects
// expected and functions to recompose them. If no function is provided then
// reflection is used instead. Panics on error.
func MustNewRecomposer(
	createKey string,
	composers map[interface{}]RecomposeFunc,
	anyComposers ...map[interface{}]RecomposeAnyFunc) *Recomposer {

	r := Recomposer{
		CreateKey: createKey,
		composers: map[string]*composer{},
	}
	for v, fun := range composers {
		rt := reflect.TypeOf(v)
		if _, err := r.registerComposer(rt, fun); err != nil {
			panic(err)
		}
	}
	if 0 < len(anyComposers) {
		for v, fun := range anyComposers[0] {
			rt := reflect.TypeOf(v)
			if _, err := r.registerAnyComposer(rt, fun); err != nil {
				panic(err)
			}
		}
	}
	return &r
}
