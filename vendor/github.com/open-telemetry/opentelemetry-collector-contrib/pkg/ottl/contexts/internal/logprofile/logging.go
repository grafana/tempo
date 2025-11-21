// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package logprofile // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/internal/logprofile"

import (
	"encoding/hex"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap/zapcore"
)

func getMapping(dict pprofile.ProfilesDictionary, idx int32) (mapping, error) {
	mTable := dict.MappingTable()
	if idx >= int32(mTable.Len()) {
		return mapping{}, fmt.Errorf("mapping index out of bounds: %d", idx)
	}
	return newMapping(dict, mTable.At(int(idx)))
}

func getLocations(dict pprofile.ProfilesDictionary, stackIDx int32) (locations, error) {
	locTable := dict.LocationTable()
	stackTable := dict.StackTable()

	var joinedErr error
	locIdxs := stackTable.At(int(stackIDx)).LocationIndices()
	ls := make(locations, 0, locIdxs.Len())
	for i := range locIdxs.Len() {
		locIdx := locIdxs.At(i)
		l, err := newLocation(dict, locTable.At(int(locIdx)))
		joinedErr = errors.Join(joinedErr, err)
		ls = append(ls, l)
	}

	return ls, joinedErr
}

func getFunction(dict pprofile.ProfilesDictionary, idx int32) (function, error) {
	fnTable := dict.FunctionTable()
	if idx >= int32(fnTable.Len()) {
		return function{}, fmt.Errorf("function index out of bounds: %d", idx)
	}
	return newFunction(dict, fnTable.At(int(idx)))
}

func getLink(dict pprofile.ProfilesDictionary, idx int32) (link, error) {
	lTable := dict.LinkTable()
	if idx >= int32(lTable.Len()) {
		return link{}, fmt.Errorf("link index out of bounds: %d", idx)
	}
	return link{lTable.At(int(idx))}, nil
}

func getString(dict pprofile.ProfilesDictionary, idx int32) (string, error) {
	strTable := dict.StringTable()
	if idx >= int32(strTable.Len()) {
		return "", fmt.Errorf("string index out of bounds: %d", idx)
	}
	return strTable.At(int(idx)), nil
}

func getAttribute(dict pprofile.ProfilesDictionary, idx int32) (attribute, error) {
	attrTable := dict.AttributeTable()
	strTable := dict.StringTable()
	if idx >= int32(attrTable.Len()) {
		return attribute{}, fmt.Errorf("attribute index out of bounds: %d", idx)
	}
	attr := attrTable.At(int(idx))
	// Is there a better way to marshal the value?
	return attribute{strTable.At(int(attr.KeyStrindex())), attr.Value().AsString()}, nil
}

type Profile struct {
	pprofile.Profile
	Dictionary pprofile.ProfilesDictionary
}

func (p Profile) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	var joinedErr error

	vts, err := newValueType(p, p.SampleType())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddObject("sample_type", vts))

	samples := p.Samples()
	for _, s := range samples.All() {
		joinedErr = errors.Join(joinedErr, encoder.AddObject("sample", ProfileSample{
			s,
			p.Profile,
			p.Dictionary,
		}))
	}

	encoder.AddInt64("time_nanos", int64(p.Time()))
	encoder.AddInt64("duration_nanos", int64(p.Duration()))

	vt, err := newValueType(p, p.PeriodType())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddObject("period_type", vt))

	encoder.AddInt64("period", p.Period())

	pid := p.ProfileID()
	encoder.AddString("profile_id", hex.EncodeToString(pid[:]))
	encoder.AddUint32("dropped_attributes_count", p.DroppedAttributesCount())
	encoder.AddString("original_payload_format", p.OriginalPayloadFormat())
	encoder.AddByteString("original_payload", p.OriginalPayload().AsRaw())

	ats, err := newAttributes(p.Dictionary, p.AttributeIndices())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddArray("attributes", ats))

	return joinedErr
}

type ProfileSample struct {
	pprofile.Sample
	Profile    pprofile.Profile
	Dictionary pprofile.ProfilesDictionary
}

func (s ProfileSample) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	var joinedErr error

	locs, err := getLocations(s.Dictionary, s.StackIndex())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddArray("locations", locs))

	values := newValues(s.Values())
	joinedErr = errors.Join(joinedErr, encoder.AddArray("values", values))

	ats, err := newAttributes(s.Dictionary, s.AttributeIndices())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddArray("attributes", ats))

	if s.LinkIndex() > 0 {
		l, err := getLink(s.Dictionary, s.LinkIndex())
		joinedErr = errors.Join(joinedErr, err)
		joinedErr = errors.Join(joinedErr, encoder.AddObject("link", l))
	}

	ts := newTimestamps(s.TimestampsUnixNano())
	joinedErr = errors.Join(joinedErr, encoder.AddArray("timestamps_unix_nano", ts))

	return joinedErr
}

type valueType struct {
	typ                    string
	unit                   string
	aggregationTemporality int32
}

func newValueType(p Profile, vt pprofile.ValueType) (valueType, error) {
	var result valueType
	var err, joinedErr error

	result.typ, err = getString(p.Dictionary, vt.TypeStrindex())
	joinedErr = errors.Join(joinedErr, err)
	result.unit, err = getString(p.Dictionary, vt.UnitStrindex())
	joinedErr = errors.Join(joinedErr, err)

	return result, joinedErr
}

func (vt valueType) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("type", vt.typ)
	encoder.AddString("unit", vt.unit)
	encoder.AddInt32("aggregation_temporality", vt.aggregationTemporality)
	return nil
}

type locations []location

func (s locations) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	var joinedErr error
	for _, l := range s {
		joinedErr = errors.Join(joinedErr, encoder.AppendObject(l))
	}
	return joinedErr
}

type location struct {
	mapping    mapping
	address    uint64
	lines      lines
	attributes attributes
}

func newLocation(dict pprofile.ProfilesDictionary, pl pprofile.Location) (location, error) {
	var l location
	var err, joinedErr error

	if pl.MappingIndex() != 0 { // optional
		if l.mapping, err = getMapping(dict, pl.MappingIndex()); err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}
	if l.attributes, err = newAttributes(dict, pl.AttributeIndices()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	if l.lines, err = newLines(dict, pl.Lines()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	l.address = pl.Address()

	return l, joinedErr
}

func (l location) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("address", l.address)
	err := encoder.AddObject("mapping", l.mapping)
	err = errors.Join(err, encoder.AddArray("lines", l.lines))
	return errors.Join(err, encoder.AddArray("attributes", l.attributes))
}

type mapping struct {
	filename    string
	memoryStart uint64
	memoryLimit uint64
	fileOffset  uint64
}

func newMapping(dict pprofile.ProfilesDictionary, pm pprofile.Mapping) (mapping, error) {
	var m mapping
	var err error

	m.filename, err = getString(dict, pm.FilenameStrindex())
	m.memoryStart = pm.MemoryStart()
	m.memoryLimit = pm.MemoryLimit()
	m.fileOffset = pm.FileOffset()

	return m, err
}

func (m mapping) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("filename", m.filename)
	encoder.AddUint64("memory_start", m.memoryStart)
	encoder.AddUint64("memory_limit", m.memoryLimit)
	encoder.AddUint64("file_offset", m.fileOffset)
	return nil
}

type link struct {
	pprofile.Link
}

func (m link) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	traceID := m.TraceID()
	encoder.AddString("trace_id", hex.EncodeToString(traceID[:]))
	spanID := m.SpanID()
	encoder.AddString("span_id", hex.EncodeToString(spanID[:]))
	return nil
}

type attributes []attribute

func (s attributes) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	var joinedErr error
	for _, a := range s {
		if err := encoder.AppendObject(a); err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}
	return joinedErr
}

func newAttributes(dict pprofile.ProfilesDictionary, pattrs pcommon.Int32Slice) (attributes, error) {
	var joinedErr error
	as := make(attributes, 0, pattrs.Len())
	for i := range pattrs.Len() {
		a, err := getAttribute(dict, pattrs.At(i))
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
		as = append(as, a)
	}
	return as, joinedErr
}

type attribute struct {
	key   string
	value string
}

func (a attribute) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("key", a.key)
	encoder.AddString("value", a.value)
	return nil
}

type lines []line

func (s lines) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	var joinedErr error
	for _, l := range s {
		if err := encoder.AppendObject(l); err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}
	return joinedErr
}

func newLines(dict pprofile.ProfilesDictionary, plines pprofile.LineSlice) (lines, error) {
	var joinedErr error
	ls := make(lines, 0, plines.Len())
	for i := range plines.Len() {
		l, err := newLine(dict, plines.At(i))
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
		ls = append(ls, l)
	}
	return ls, joinedErr
}

type line struct {
	function function
	line     int64
	column   int64
}

func newLine(dict pprofile.ProfilesDictionary, pl pprofile.Line) (line, error) {
	var l line
	var err, joinedErr error

	if l.function, err = getFunction(dict, pl.FunctionIndex()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	l.line = pl.Line()
	l.column = pl.Column()

	return l, joinedErr
}

func (l line) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddInt64("line", l.line)
	encoder.AddInt64("column", l.column)
	return encoder.AddObject("function", l.function)
}

type function struct {
	name       string
	systemName string
	filename   string
	startLine  int64
}

func newFunction(dict pprofile.ProfilesDictionary, pf pprofile.Function) (function, error) {
	var f function
	var err, joinedErr error

	if f.name, err = getString(dict, pf.NameStrindex()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	if f.systemName, err = getString(dict, pf.SystemNameStrindex()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	if f.filename, err = getString(dict, pf.FilenameStrindex()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	f.startLine = pf.StartLine()

	return f, joinedErr
}

func (f function) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("name", f.name)
	encoder.AddString("system_name", f.systemName)
	encoder.AddString("filename", f.filename)
	encoder.AddInt64("start_line", f.startLine)
	return nil
}

type timestamps []timestamp

func (s timestamps) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	var joinedErr error
	for _, t := range s {
		if err := encoder.AppendObject(t); err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}
	return joinedErr
}

func newTimestamps(ptimestamps pcommon.UInt64Slice) timestamps {
	ts := make(timestamps, 0, ptimestamps.Len())
	for i := range ptimestamps.Len() {
		ts = append(ts, timestamp(ptimestamps.At(i)))
	}
	return ts
}

type timestamp uint64

func (l timestamp) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("timestamp_unix_nano", uint64(l))
	return nil
}

type values []value

func (s values) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	var joinedErr error
	for _, v := range s {
		if err := encoder.AppendObject(v); err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}
	return joinedErr
}

func newValues(pvalues pcommon.Int64Slice) values {
	vs := make(values, 0, pvalues.Len())
	for i := range pvalues.Len() {
		vs = append(vs, value(pvalues.At(i)))
	}
	return vs
}

type value int64

func (v value) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddInt64("value", int64(v))
	return nil
}
