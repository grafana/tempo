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

type Profile pprofile.Profile

func (p Profile) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	pp := pprofile.Profile(p)
	var joinedErr error

	vts, err := newValueTypes(p, pp.SampleType())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddArray("sample_type", vts))

	ss, err := newSamples(p, pp.Sample())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddArray("sample", ss))

	encoder.AddInt64("time_nanos", int64(pp.Time()))
	encoder.AddInt64("duration_nanos", int64(pp.Duration()))

	vt, err := newValueType(p, pp.PeriodType())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddObject("period_type", vt))

	encoder.AddInt64("period", pp.Period())

	cs, err := p.getComments()
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddArray("comments", cs))

	dst, err := p.getString(pp.DefaultSampleTypeStrindex())
	joinedErr = errors.Join(joinedErr, err)
	encoder.AddString("default_sample_type", dst)

	pid := pp.ProfileID()
	encoder.AddString("profile_id", hex.EncodeToString(pid[:]))
	encoder.AddUint32("dropped_attributes_count", pp.DroppedAttributesCount())
	encoder.AddString("original_payload_format", pp.OriginalPayloadFormat())
	encoder.AddByteString("original_payload", pp.OriginalPayload().AsRaw())

	ats, err := newAttributes(p, pp.AttributeIndices())
	joinedErr = errors.Join(joinedErr, err)
	joinedErr = errors.Join(joinedErr, encoder.AddArray("attributes", ats))

	return joinedErr
}

func (p Profile) getString(idx int32) (string, error) {
	pp := pprofile.Profile(p)
	strTable := pp.StringTable()
	if idx >= int32(strTable.Len()) {
		return "", fmt.Errorf("string index out of bounds: %d", idx)
	}
	return strTable.At(int(idx)), nil
}

func (p Profile) getFunction(idx int32) (function, error) {
	pp := pprofile.Profile(p)
	fnTable := pp.FunctionTable()
	if idx >= int32(fnTable.Len()) {
		return function{}, fmt.Errorf("function index out of bounds: %d", idx)
	}
	return newFunction(p, fnTable.At(int(idx)))
}

func (p Profile) getMapping(idx int32) (mapping, error) {
	pp := pprofile.Profile(p)
	mTable := pp.MappingTable()
	if idx >= int32(mTable.Len()) {
		return mapping{}, fmt.Errorf("mapping index out of bounds: %d", idx)
	}
	return newMapping(p, mTable.At(int(idx)))
}

func (p Profile) getLink(idx int32) (link, error) {
	pp := pprofile.Profile(p)
	lTable := pp.LinkTable()
	if idx >= int32(lTable.Len()) {
		return link{}, fmt.Errorf("link index out of bounds: %d", idx)
	}
	return link{lTable.At(int(idx))}, nil
}

func (p Profile) getLocations(start, length int32) (locations, error) {
	pp := pprofile.Profile(p)
	locTable := pp.LocationTable()
	if start >= int32(locTable.Len()) {
		return locations{}, fmt.Errorf("location start index out of bounds: %d", start)
	}
	if start+length > int32(locTable.Len()) {
		return locations{}, fmt.Errorf("location end index out of bounds: %d", start+length)
	}

	var joinedErr error
	ls := make(locations, 0, length)
	for i := range length {
		l, err := newLocation(p, locTable.At(int(start+i)))
		joinedErr = errors.Join(joinedErr, err)
		ls = append(ls, l)
	}

	return ls, joinedErr
}

func (p Profile) getAttribute(idx int32) (attribute, error) {
	pp := pprofile.Profile(p)
	attrTable := pp.AttributeTable()
	if idx >= int32(attrTable.Len()) {
		return attribute{}, fmt.Errorf("attribute index out of bounds: %d", idx)
	}
	attr := attrTable.At(int(idx))
	// Is there a better way to marshal the value?
	return attribute{attr.Key(), attr.Value().AsString()}, nil
}

type samples []sample

func (ss samples) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	var joinedErr error
	for _, s := range ss {
		joinedErr = errors.Join(joinedErr, encoder.AppendObject(s))
	}
	return joinedErr
}

func newSamples(p Profile, sampleSlice pprofile.SampleSlice) (samples, error) {
	var joinedErr error
	ss := make(samples, 0, sampleSlice.Len())
	for i := range sampleSlice.Len() {
		s, err := newSample(p, sampleSlice.At(i))
		joinedErr = errors.Join(joinedErr, err)
		ss = append(ss, s)
	}
	return ss, joinedErr
}

type sample struct {
	timestamps timestamps
	attributes attributes
	locations  locations
	values     values
	link       *link
}

func newSample(p Profile, ps pprofile.Sample) (sample, error) {
	var s sample
	var err, joinedErr error

	s.timestamps = newTimestamps(ps.TimestampsUnixNano())
	s.values = newValues(ps.Value())
	s.attributes, err = newAttributes(p, ps.AttributeIndices())
	joinedErr = errors.Join(joinedErr, err)
	s.locations, err = p.getLocations(ps.LocationsStartIndex(), ps.LocationsLength())
	joinedErr = errors.Join(joinedErr, err)
	if ps.HasLinkIndex() { // optional
		l, err := p.getLink(ps.LinkIndex())
		joinedErr = errors.Join(joinedErr, err)
		s.link = &l
	}

	return s, joinedErr
}

func (s sample) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	err := encoder.AddArray("timestamps_unix_nano", s.timestamps)
	err = errors.Join(err, encoder.AddArray("attributes", s.attributes))
	err = errors.Join(err, encoder.AddArray("locations", s.locations))
	return errors.Join(err, encoder.AddArray("values", s.values))
}

type valueTypes []valueType

func (s valueTypes) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	var err error
	for _, vt := range s {
		err = errors.Join(err, encoder.AppendObject(vt))
	}
	return err
}

func newValueTypes(p Profile, sampleTypes pprofile.ValueTypeSlice) (valueTypes, error) {
	var joinedErr error

	vts := make(valueTypes, 0, sampleTypes.Len())
	for i := range sampleTypes.Len() {
		vt, err := newValueType(p, sampleTypes.At(i))
		joinedErr = errors.Join(joinedErr, err)
		vts = append(vts, vt)
	}

	return vts, joinedErr
}

type valueType struct {
	typ                    string
	unit                   string
	aggregationTemporality int32
}

func newValueType(p Profile, vt pprofile.ValueType) (valueType, error) {
	var result valueType
	var err, joinedErr error

	result.aggregationTemporality = int32(vt.AggregationTemporality())
	result.typ, err = p.getString(vt.TypeStrindex())
	joinedErr = errors.Join(joinedErr, err)
	result.unit, err = p.getString(vt.UnitStrindex())
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
	isFolded   bool
	attributes attributes
}

func newLocation(p Profile, pl pprofile.Location) (location, error) {
	var l location
	var err, joinedErr error

	if pl.MappingIndex() != 0 { // optional
		if l.mapping, err = p.getMapping(pl.MappingIndex()); err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}
	if l.attributes, err = newAttributes(p, pl.AttributeIndices()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	if l.lines, err = newLines(p, pl.Line()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	l.address = pl.Address()
	l.isFolded = pl.IsFolded()

	return l, joinedErr
}

func (l location) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddUint64("address", l.address)
	encoder.AddBool("is_folded", l.isFolded)
	err := encoder.AddObject("mapping", l.mapping)
	err = errors.Join(err, encoder.AddArray("lines", l.lines))
	return errors.Join(err, encoder.AddArray("attributes", l.attributes))
}

type mapping struct {
	filename        string
	memoryStart     uint64
	memoryLimit     uint64
	fileOffset      uint64
	hasFunctions    bool
	hasFilenames    bool
	hasLineNumbers  bool
	hasInlineFrames bool
}

func newMapping(p Profile, pm pprofile.Mapping) (mapping, error) {
	var m mapping
	var err error

	m.filename, err = p.getString(pm.FilenameStrindex())
	m.memoryStart = pm.MemoryStart()
	m.memoryLimit = pm.MemoryLimit()
	m.fileOffset = pm.FileOffset()
	m.hasFunctions = pm.HasFunctions()
	m.hasFilenames = pm.HasFilenames()
	m.hasLineNumbers = pm.HasLineNumbers()
	m.hasInlineFrames = pm.HasInlineFrames()

	return m, err
}

func (m mapping) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("filename", m.filename)
	encoder.AddUint64("memory_start", m.memoryStart)
	encoder.AddUint64("memory_limit", m.memoryLimit)
	encoder.AddUint64("file_offset", m.fileOffset)
	encoder.AddBool("has_functions", m.hasFunctions)
	encoder.AddBool("has_filenames", m.hasFilenames)
	encoder.AddBool("has_line_numbers", m.hasLineNumbers)
	encoder.AddBool("has_inline_frames", m.hasInlineFrames)
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

func newAttributes(p Profile, pattrs pcommon.Int32Slice) (attributes, error) {
	var joinedErr error
	as := make(attributes, 0, pattrs.Len())
	for i := range pattrs.Len() {
		a, err := p.getAttribute(pattrs.At(i))
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

func newLines(p Profile, plines pprofile.LineSlice) (lines, error) {
	var joinedErr error
	ls := make(lines, 0, plines.Len())
	for i := range plines.Len() {
		l, err := newLine(p, plines.At(i))
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

func newLine(p Profile, pl pprofile.Line) (line, error) {
	var l line
	var err, joinedErr error

	if l.function, err = p.getFunction(pl.FunctionIndex()); err != nil {
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

func newFunction(p Profile, pf pprofile.Function) (function, error) {
	var f function
	var err, joinedErr error

	if f.name, err = p.getString(pf.NameStrindex()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	if f.systemName, err = p.getString(pf.SystemNameStrindex()); err != nil {
		joinedErr = errors.Join(joinedErr, err)
	}
	if f.filename, err = p.getString(pf.FilenameStrindex()); err != nil {
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

type comments []string

func (p Profile) getComments() (comments, error) {
	var joinedErr error
	pp := pprofile.Profile(p)
	l := pp.CommentStrindices().Len()
	cs := make(comments, 0, l)
	for i := range l {
		c, err := p.getString(pp.CommentStrindices().At(i))
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
		cs = append(cs, c)
	}
	return cs, joinedErr
}

func (cs comments) MarshalLogArray(encoder zapcore.ArrayEncoder) error {
	for _, s := range cs {
		encoder.AppendString(s)
	}
	return nil
}
