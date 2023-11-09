package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
	"github.com/grafana/tempo/tempodb/encoding/vparquet4"
)

type convertParquet3to4V2 struct {
	In  string `arg:"" help:"The input parquet file to read from"`
	Out string `arg:"" help:"The output parquet file to write to"`
}

func (cmd *convertParquet3to4V2) Run() error {
	err := cmd.convert()
	if err != nil {
		return err
	}

	return cmd.printFileInfo()
}

func (cmd *convertParquet3to4V2) printFileInfo() error {
	fmt.Printf("Converted file: %s\n", cmd.Out)
	out, err := os.Open(cmd.Out)
	if err != nil {
		return err
	}
	defer out.Close()

	stat, err := out.Stat()
	if err != nil {
		return err
	}
	fmt.Printf("File size: %d\n", stat.Size())

	buf := make([]byte, 8)
	n, err := out.ReadAt(buf, stat.Size()-8)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if n < 4 {
		return errors.New("not enough bytes read to determine footer size")
	}
	fmt.Printf("Footer size: %d\n", binary.LittleEndian.Uint32(buf[0:4]))

	return nil
}

func (cmd *convertParquet3to4V2) convert() error {
	in, err := os.Open(cmd.In)
	if err != nil {
		return err
	}
	defer in.Close()

	inStat, err := in.Stat()
	if err != nil {
		return err
	}

	pf, err := parquet.OpenFile(in, inStat.Size())
	if err != nil {
		return err
	}

	out, err := os.OpenFile(cmd.Out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, inStat.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	writer := parquet.NewGenericWriter[vparquet4.Trace](out)
	defer writer.Close()

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
				return err
			}
			if readCount == 0 {
				err = writer.Flush()
				if err != nil {
					return err
				}
				break
			}

			vparquet3to4(readBuffer[:readCount], writeBuffer)

			writeCount := 0
			for writeCount < readCount {
				n, err := writer.Write(writeBuffer[writeCount:readCount])
				if err != nil {
					return err
				}
				writeCount += n
			}
		}
	}

	return nil
}

func vparquet3to4(in []vparquet3.Trace, out []vparquet4.Trace) {
	for i, trace := range in {
		vparquetTrace3to4(&trace, &out[i])
	}
}

func vparquetTrace3to4(trace *vparquet3.Trace, v4trace *vparquet4.Trace) {
	v4trace.TraceID = trace.TraceID
	v4trace.TraceIDText = trace.TraceIDText
	v4trace.ResourceSpans = make([]vparquet4.ResourceSpans, len(trace.ResourceSpans))
	v4trace.StartTimeUnixNano = trace.StartTimeUnixNano
	v4trace.EndTimeUnixNano = trace.EndTimeUnixNano
	v4trace.DurationNano = trace.DurationNano
	v4trace.RootServiceName = trace.RootServiceName
	v4trace.RootSpanName = trace.RootSpanName

	for i, rspan := range trace.ResourceSpans {
		vparquetResourceSpans3to4(&rspan, &v4trace.ResourceSpans[i])
	}
}

func vparquetResourceSpans3to4(rspan *vparquet3.ResourceSpans, v4rspan *vparquet4.ResourceSpans) {
	v4rspan.Resource.ServiceName = rspan.Resource.ServiceName
	v4rspan.Resource.Cluster = rspan.Resource.Cluster
	v4rspan.Resource.Namespace = rspan.Resource.Namespace
	v4rspan.Resource.Pod = rspan.Resource.Pod
	v4rspan.Resource.Container = rspan.Resource.Container
	v4rspan.Resource.K8sClusterName = rspan.Resource.K8sClusterName
	v4rspan.Resource.K8sNamespaceName = rspan.Resource.K8sNamespaceName
	v4rspan.Resource.K8sPodName = rspan.Resource.K8sPodName
	v4rspan.Resource.K8sContainerName = rspan.Resource.K8sContainerName

	v4rspan.Resource.Attrs = make([]vparquet4.ArrayAttribute, len(rspan.Resource.Attrs))
	for i, attr := range rspan.Resource.Attrs {
		vparquetAttribute3to4(attr, &v4rspan.Resource.Attrs[i])
	}

	v4rspan.ScopeSpans = make([]vparquet4.ScopeSpans, len(rspan.ScopeSpans))
	for i, sspan := range rspan.ScopeSpans {
		v4sspan := &v4rspan.ScopeSpans[i]
		v4sspan.Scope.Name = sspan.Scope.Name
		v4sspan.Scope.Version = sspan.Scope.Version

		v4sspan.Spans = make([]vparquet4.Span, len(sspan.Spans))
		for j, span := range sspan.Spans {
			vparquetSpan3to4(&span, &v4sspan.Spans[j])
		}
	}
}

func vparquetSpan3to4(span *vparquet3.Span, v4span *vparquet4.Span) {
	v4span.SpanID = span.SpanID
	v4span.Name = span.Name
	v4span.Kind = span.Kind
	v4span.ParentSpanID = span.ParentSpanID
	v4span.TraceState = span.TraceState
	v4span.StartTimeUnixNano = span.StartTimeUnixNano
	v4span.DurationNano = span.DurationNano
	v4span.StatusCode = span.StatusCode
	v4span.StatusMessage = span.StatusMessage
	v4span.DroppedAttributesCount = span.DroppedAttributesCount
	v4span.DroppedEventsCount = span.DroppedEventsCount
	v4span.Links = span.Links
	v4span.DroppedLinksCount = span.DroppedLinksCount
	v4span.HttpMethod = span.HttpMethod
	v4span.HttpUrl = span.HttpUrl
	v4span.HttpStatusCode = span.HttpStatusCode

	v4span.Events = make([]vparquet4.Event, len(span.Events))
	for i, event := range span.Events {
		vparquetEvent3to4(&event, &v4span.Events[i])
	}

	v4span.Attrs = make([]vparquet4.ArrayAttribute, 0, len(span.Attrs))
	for _, attr := range span.Attrs {
		var v4attr vparquet4.ArrayAttribute
		vparquetAttribute3to4(attr, &v4attr)
		v4span.Attrs = append(v4span.Attrs, v4attr)
	}
}

func vparquetAttribute3to4(attr vparquet3.Attribute, v4attr *vparquet4.ArrayAttribute) {
	v4attr.Key = attr.Key
	if attr.Value != nil {
		v4attr.Value = []string{*attr.Value}
	}

	if attr.ValueInt != nil {
		v4attr.ValueInt = []int64{*attr.ValueInt}
	}

	if attr.ValueDouble != nil {
		v4attr.ValueDouble = []float64{*attr.ValueDouble}
	}

	if attr.ValueBool != nil {
		v4attr.ValueBool = []bool{*attr.ValueBool}
	}

	v4attr.ValueKVList = attr.ValueKVList
	v4attr.ValueArray = attr.ValueArray
}

func vparquetEvent3to4(event *vparquet3.Event, v4event *vparquet4.Event) {
	v4event.TimeUnixNano = event.TimeUnixNano
	v4event.Name = event.Name
	v4event.DroppedAttributesCount = event.DroppedAttributesCount

	v4event.Attrs = make([]vparquet4.EventAttribute, len(event.Attrs))
	for i, attr := range event.Attrs {
		v4attr := &v4event.Attrs[i]
		v4attr.Key = attr.Key
		v4attr.Value = attr.Value
	}
}
