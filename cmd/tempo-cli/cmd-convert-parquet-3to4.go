package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	v1 "github.com/grafana/tempo/pkg/tempopb/common/v1"

	"github.com/grafana/tempo/pkg/tempopb"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

const (
	blockFilename = "data.parquet"
	metaFilename  = "meta.json"
)

type convertParquet3to4 struct {
	In  string `arg:"" help:"The input parquet file to read from"`
	Out string `arg:"" help:"The output parquet file to write to"`
}

func (cmd *convertParquet3to4) Run() error {
	meta, err := cmd.convert()
	if err != nil {
		return err
	}

	cmd.printFileInfo(meta)
	return nil
}

func (cmd *convertParquet3to4) printFileInfo(meta *backend.BlockMeta) {
	fmt.Printf("Converted file: %s\n", filepath.Join(cmd.Out, blockFilename))
	fmt.Printf("File size: %d\n", meta.Size)
	fmt.Printf("Footer size: %d\n", meta.FooterSize)
}

func (cmd *convertParquet3to4) convert() (*backend.BlockMeta, error) {
	inMeta, err := cmd.readBlockMeta()
	if err != nil {
		return nil, err
	}

	// open old data.parquet
	in, err := os.Open(filepath.Join(cmd.In, blockFilename))
	if err != nil {
		return nil, err
	}
	defer in.Close()

	pf, err := parquet.OpenFile(in, int64(inMeta.Size))
	if err != nil {
		return nil, err
	}

	// open new data.parquet
	inStat, err := in.Stat()
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(cmd.Out, 0o755)
	if err != nil {
		return nil, err
	}
	out, err := os.OpenFile(filepath.Join(cmd.Out, blockFilename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, inStat.Mode())
	if err != nil {
		return nil, err
	}
	defer out.Close()

	writer := parquet.NewGenericWriter[vparquet4.Trace](out)

	readBuffer := make([]vparquet3.Trace, 500)
	writeBuffer := make([]vparquet4.Trace, 500)

	rowGroups := pf.RowGroups()
	fmt.Printf("Total rowgroups: %d\n", len(rowGroups))

	for i, rowGroup := range rowGroups {
		fmt.Printf("Converting rowgroup: %d\n", i+1)
		reader := parquet.NewGenericRowGroupReader[vparquet3.Trace](rowGroup)

		for {
			readCount, err := reader.Read(readBuffer)
			if err != nil && !errors.Is(err, io.EOF) {
				return nil, err
			}
			if readCount == 0 {
				err = writer.Flush()
				if err != nil {
					return nil, err
				}
				break
			}

			err = vparquet3to4(readBuffer[:readCount], writeBuffer)
			if err != nil {
				return nil, err
			}

			writeCount := 0
			for writeCount < readCount {
				n, err := writer.Write(writeBuffer[writeCount:readCount])
				if err != nil {
					return nil, err
				}
				writeCount += n
			}
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	// convert and write meta.json
	return cmd.writeBlockMetaV4(inMeta)
}

func (cmd *convertParquet3to4) readBlockMeta() (*backend.BlockMeta, error) {
	inMeta, err := os.Open(filepath.Join(cmd.In, metaFilename))
	if err != nil {
		return nil, err
	}

	var meta backend.BlockMeta
	err = json.NewDecoder(inMeta).Decode(&meta)
	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (cmd *convertParquet3to4) writeBlockMetaV4(meta *backend.BlockMeta) (*backend.BlockMeta, error) {
	out, err := os.Open(filepath.Join(cmd.Out, blockFilename))
	if err != nil {
		return nil, err
	}
	defer out.Close()

	// read file size
	stat, err := out.Stat()
	if err != nil {
		return nil, err
	}

	metaV4 := *meta
	metaV4.Version = vparquet4.VersionString
	metaV4.Size = uint64(stat.Size())

	// read footer size
	buf := make([]byte, 8)
	n, err := out.ReadAt(buf, stat.Size()-8)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if n < 4 {
		return nil, errors.New("not enough bytes read to determine footer size")
	}
	metaV4.FooterSize = binary.LittleEndian.Uint32(buf[0:4])

	// write vParquet4 meta
	outMeta, err := os.OpenFile(filepath.Join(cmd.Out, metaFilename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, stat.Mode())
	if err != nil {
		return nil, err
	}
	defer outMeta.Close()

	err = json.NewEncoder(outMeta).Encode(&metaV4)
	if err != nil {
		return nil, err
	}

	return &metaV4, nil
}

func vparquet3to4(in []vparquet3.Trace, out []vparquet4.Trace) error {
	for i, trace := range in {
		err := vparquetTrace3to4(&trace, &out[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func vparquetTrace3to4(trace *vparquet3.Trace, v4trace *vparquet4.Trace) error {
	v4trace.TraceID = trace.TraceID
	v4trace.TraceIDText = trace.TraceIDText
	v4trace.StartTimeUnixNano = trace.StartTimeUnixNano
	v4trace.EndTimeUnixNano = trace.EndTimeUnixNano
	v4trace.DurationNano = trace.DurationNano
	v4trace.RootServiceName = trace.RootServiceName
	v4trace.RootSpanName = trace.RootSpanName

	v4trace.ResourceSpans = make([]vparquet4.ResourceSpans, len(trace.ResourceSpans))
	for i, rspan := range trace.ResourceSpans {
		err := vparquetResourceSpans3to4(&rspan, &v4trace.ResourceSpans[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func vparquetResourceSpans3to4(rspan *vparquet3.ResourceSpans, v4rspan *vparquet4.ResourceSpans) error {
	v4rspan.Resource.ServiceName = rspan.Resource.ServiceName
	v4rspan.Resource.Cluster = rspan.Resource.Cluster
	v4rspan.Resource.Namespace = rspan.Resource.Namespace
	v4rspan.Resource.Pod = rspan.Resource.Pod
	v4rspan.Resource.Container = rspan.Resource.Container
	v4rspan.Resource.K8sClusterName = rspan.Resource.K8sClusterName
	v4rspan.Resource.K8sNamespaceName = rspan.Resource.K8sNamespaceName
	v4rspan.Resource.K8sPodName = rspan.Resource.K8sPodName
	v4rspan.Resource.K8sContainerName = rspan.Resource.K8sContainerName

	vparquetDedicatedAttribute3to4(&rspan.Resource.DedicatedAttributes, &v4rspan.Resource.DedicatedAttributes)

	v4rspan.Resource.Attrs = make([]vparquet4.Attribute, len(rspan.Resource.Attrs))
	for i, attr := range rspan.Resource.Attrs {
		dropped := vparquetAttribute3to4(&attr, &v4rspan.Resource.Attrs[i])
		v4rspan.Resource.DroppedAttributesCount += dropped
	}

	v4rspan.ScopeSpans = make([]vparquet4.ScopeSpans, len(rspan.ScopeSpans))
	for i, sspan := range rspan.ScopeSpans {
		v4sspan := &v4rspan.ScopeSpans[i]
		v4sspan.Scope.Name = sspan.Scope.Name
		v4sspan.Scope.Version = sspan.Scope.Version

		v4sspan.Spans = make([]vparquet4.Span, len(sspan.Spans))
		for j, span := range sspan.Spans {
			err := vparquetSpan3to4(&span, &v4sspan.Spans[j])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func vparquetSpan3to4(span *vparquet3.Span, v4span *vparquet4.Span) error {
	v4span.SpanID = span.SpanID
	v4span.ParentSpanID = span.ParentSpanID
	v4span.ParentID = span.ParentID
	v4span.NestedSetLeft = span.NestedSetLeft
	v4span.NestedSetRight = span.NestedSetRight
	v4span.Name = span.Name
	v4span.Kind = span.Kind
	v4span.TraceState = span.TraceState
	v4span.StartTimeUnixNano = span.StartTimeUnixNano
	v4span.DurationNano = span.DurationNano
	v4span.StatusCode = span.StatusCode
	v4span.StatusMessage = span.StatusMessage
	v4span.HttpMethod = span.HttpMethod
	v4span.HttpUrl = span.HttpUrl
	v4span.HttpStatusCode = span.HttpStatusCode

	v4span.DroppedLinksCount = span.DroppedLinksCount
	v4links, err := vparquetLinks3to4(span.Links)
	if err != nil {
		return err
	}
	v4span.Links = v4links

	v4span.DroppedEventsCount = span.DroppedEventsCount
	v4span.Events = make([]vparquet4.Event, len(span.Events))
	for i, event := range span.Events {
		err = vparquetEvent3to4(&event, &v4span.Events[i], span.StartTimeUnixNano)
		if err != nil {
			return err
		}
	}

	vparquetDedicatedAttribute3to4(&span.DedicatedAttributes, &v4span.DedicatedAttributes)
	v4span.DroppedAttributesCount = span.DroppedAttributesCount
	v4span.Attrs = make([]vparquet4.Attribute, 0, len(span.Attrs))
	for _, attr := range span.Attrs {
		var v4attr vparquet4.Attribute
		dropped := vparquetAttribute3to4(&attr, &v4attr)
		v4span.Attrs = append(v4span.Attrs, v4attr)
		v4span.DroppedAttributesCount += dropped
	}

	return nil
}

func vparquetDedicatedAttribute3to4(attr *vparquet3.DedicatedAttributes, v4attr *vparquet4.DedicatedAttributes) {
	v4attr.String01 = attr.String01
	v4attr.String02 = attr.String02
	v4attr.String03 = attr.String03
	v4attr.String04 = attr.String04
	v4attr.String05 = attr.String05
	v4attr.String06 = attr.String06
	v4attr.String07 = attr.String07
	v4attr.String08 = attr.String08
	v4attr.String09 = attr.String09
	v4attr.String10 = attr.String10
}

const (
	attrTypeNotSupported vparquet4.AttrType = iota
	attrTypeString
	attrTypeInt
	attrTypeDouble
	attrTypeBool
)

func vparquetAttribute3to4(attr *vparquet3.Attribute, v4attr *vparquet4.Attribute) int32 {
	v4attr.Key = attr.Key
	var dropped int32

	if attr.Value != nil {
		v4attr.Value = []string{*attr.Value}
		v4attr.ValueType = attrTypeString
	} else if attr.ValueInt != nil {
		v4attr.ValueInt = []int64{*attr.ValueInt}
		v4attr.ValueType = attrTypeInt
	} else if attr.ValueDouble != nil {
		v4attr.ValueDouble = []float64{*attr.ValueDouble}
		v4attr.ValueType = attrTypeDouble
	} else if attr.ValueBool != nil {
		v4attr.ValueBool = []bool{*attr.ValueBool}
		v4attr.ValueType = attrTypeBool
	} else if attr.ValueArray != "" {
		v4attr.ValueDropped = attr.ValueArray
		dropped++
	} else if attr.ValueKVList != "" {
		v4attr.ValueDropped = attr.ValueKVList
		dropped++
	}

	return dropped
}

func vparquetEvent3to4(event *vparquet3.Event, v4event *vparquet4.Event, startTimeNano uint64) error {
	v4event.TimeSinceStartNano = event.TimeUnixNano - startTimeNano
	v4event.Name = event.Name
	v4event.DroppedAttributesCount = event.DroppedAttributesCount

	if len(event.Attrs) > 0 {
		v4event.Attrs = make([]vparquet4.Attribute, 0, len(event.Attrs))
		for _, attr := range event.Attrs {
			var v4attr vparquet4.Attribute

			var val v1.AnyValue
			err := val.Unmarshal(attr.Value)
			if err != nil {
				return fmt.Errorf("unable to unmarshal event attribute value: %w", err)
			}
			vparquet4.AttrToParquet(&v1.KeyValue{Key: attr.Key, Value: &val}, &v4attr, v4event)

			if v4event.Name == "" && v4attr.Key == "event" && v4attr.ValueType == attrTypeString && len(v4attr.Value) == 1 {
				v4event.Name = v4attr.Value[0]
				continue
			}

			v4event.Attrs = append(v4event.Attrs, v4attr)
		}

		if len(v4event.Attrs) == 0 {
			v4event.Attrs = nil
		}
	}

	return nil
}

func vparquetLinks3to4(linksBytes []byte) ([]vparquet4.Link, error) {
	if len(linksBytes) == 0 {
		return nil, nil
	}

	var links tempopb.LinkSlice
	err := links.Unmarshal(linksBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal span links: %w", err)
	}

	v4links := make([]vparquet4.Link, len(links.Links))
	for i, link := range links.Links {
		v4link := &v4links[i]
		v4link.TraceID = link.TraceId
		v4link.SpanID = link.SpanId
		v4link.TraceState = link.TraceState
		if len(link.Attributes) > 0 {
			v4link.Attrs = make([]vparquet4.Attribute, len(link.Attributes))
			for j, attr := range link.Attributes {
				vparquet4.AttrToParquet(attr, &v4link.Attrs[j], v4link)
			}
		}
	}

	return v4links, nil
}
