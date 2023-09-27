package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/parquet-go/parquet-go"

	"github.com/grafana/tempo/tempodb/encoding/vparquet"
	"github.com/grafana/tempo/tempodb/encoding/vparquet2"
)

type convertParquet1to2 struct {
	In  string `arg:"" help:"The input parquet file to read from"`
	Out string `arg:"" help:"The output parquet file to write to"`
}

func (cmd *convertParquet1to2) Run() error {
	err := cmd.convert()
	if err != nil {
		return err
	}

	return cmd.printFileInfo()
}

func (cmd *convertParquet1to2) printFileInfo() error {
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

func (cmd *convertParquet1to2) convert() error {
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

	writer := parquet.NewGenericWriter[vparquet2.Trace](out)
	defer writer.Close()

	readBuffer := make([]vparquet.Trace, 500)
	writeBuffer := make([]vparquet2.Trace, 500)

	rowGroups := pf.RowGroups()
	fmt.Printf("Total rowgroups: %d\n", len(rowGroups))

	for i, rowGroup := range rowGroups {
		fmt.Printf("Converting rowgroup: %d\n", i+1)
		reader := parquet.NewGenericRowGroupReader[vparquet.Trace](rowGroup)

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

			vparquet1to2(readBuffer[:readCount], writeBuffer)

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

func vparquet1to2(in []vparquet.Trace, out []vparquet2.Trace) {
	for i, trace := range in {
		vparquetTrace1to2(&trace, &out[i])
	}
}

func vparquetTrace1to2(trace *vparquet.Trace, v2trace *vparquet2.Trace) {
	v2trace.TraceID = trace.TraceID
	v2trace.TraceIDText = trace.TraceIDText
	v2trace.ResourceSpans = make([]vparquet2.ResourceSpans, len(trace.ResourceSpans))
	v2trace.StartTimeUnixNano = trace.StartTimeUnixNano
	v2trace.EndTimeUnixNano = trace.EndTimeUnixNano
	v2trace.DurationNano = trace.DurationNanos
	v2trace.RootServiceName = trace.RootServiceName
	v2trace.RootSpanName = trace.RootSpanName

	for i, rspan := range trace.ResourceSpans {
		vparquetResourceSpans1to2(&rspan, &v2trace.ResourceSpans[i])
	}
}

func vparquetResourceSpans1to2(rspan *vparquet.ResourceSpans, v2rspan *vparquet2.ResourceSpans) {
	v2rspan.Resource.ServiceName = rspan.Resource.ServiceName
	v2rspan.Resource.Cluster = rspan.Resource.Cluster
	v2rspan.Resource.Namespace = rspan.Resource.Namespace
	v2rspan.Resource.Pod = rspan.Resource.Pod
	v2rspan.Resource.Container = rspan.Resource.Container
	v2rspan.Resource.K8sClusterName = rspan.Resource.K8sClusterName
	v2rspan.Resource.K8sNamespaceName = rspan.Resource.K8sNamespaceName
	v2rspan.Resource.K8sPodName = rspan.Resource.K8sPodName
	v2rspan.Resource.K8sContainerName = rspan.Resource.K8sContainerName

	v2rspan.Resource.Test = rspan.Resource.Test

	v2rspan.Resource.Attrs = make([]vparquet2.Attribute, len(rspan.Resource.Attrs))
	for i, attr := range rspan.Resource.Attrs {
		vparquetAttribute1to2(&attr, &v2rspan.Resource.Attrs[i])
	}

	v2rspan.ScopeSpans = make([]vparquet2.ScopeSpans, len(rspan.ScopeSpans))
	for i, sspan := range rspan.ScopeSpans {
		v2sspan := &v2rspan.ScopeSpans[i]
		v2sspan.Scope.Name = sspan.Scope.Name
		v2sspan.Scope.Version = sspan.Scope.Version

		v2sspan.Spans = make([]vparquet2.Span, len(sspan.Spans))
		for j, span := range sspan.Spans {
			vparquetSpan1to2(&span, &v2sspan.Spans[j])
		}
	}
}

func vparquetSpan1to2(span *vparquet.Span, v2span *vparquet2.Span) {
	v2span.SpanID = span.ID
	v2span.Name = span.Name
	v2span.Kind = span.Kind
	v2span.ParentSpanID = span.ParentSpanID
	v2span.TraceState = span.TraceState
	v2span.StartTimeUnixNano = span.StartUnixNanos
	v2span.DurationNano = span.EndUnixNanos - span.StartUnixNanos
	v2span.StatusCode = span.StatusCode
	v2span.StatusMessage = span.StatusMessage
	v2span.DroppedAttributesCount = span.DroppedAttributesCount
	v2span.DroppedEventsCount = span.DroppedEventsCount
	v2span.Links = span.Links
	v2span.DroppedLinksCount = span.DroppedLinksCount
	v2span.HttpMethod = span.HttpMethod
	v2span.HttpUrl = span.HttpUrl
	v2span.HttpStatusCode = span.HttpStatusCode

	v2span.Events = make([]vparquet2.Event, len(span.Events))
	for i, event := range span.Events {
		vparquetEvent1to2(&event, &v2span.Events[i])
	}

	v2span.Attrs = make([]vparquet2.Attribute, 0, len(span.Attrs))
	for _, attr := range span.Attrs {
		var v2attr vparquet2.Attribute
		vparquetAttribute1to2(&attr, &v2attr)
		v2span.Attrs = append(v2span.Attrs, v2attr)
	}
}

func vparquetAttribute1to2(attr *vparquet.Attribute, v2attr *vparquet2.Attribute) {
	v2attr.Key = attr.Key
	v2attr.Value = attr.Value
	v2attr.ValueInt = attr.ValueInt
	v2attr.ValueDouble = attr.ValueDouble
	v2attr.ValueBool = attr.ValueBool
	v2attr.ValueKVList = attr.ValueKVList
	v2attr.ValueArray = attr.ValueArray
}

func vparquetEvent1to2(event *vparquet.Event, v2event *vparquet2.Event) {
	v2event.TimeUnixNano = event.TimeUnixNano
	v2event.Name = event.Name
	v2event.DroppedAttributesCount = event.DroppedAttributesCount

	v2event.Test = event.Test

	v2event.Attrs = make([]vparquet2.EventAttribute, len(event.Attrs))
	for i, attr := range event.Attrs {
		v2attr := &v2event.Attrs[i]
		v2attr.Key = attr.Key
		v2attr.Value = attr.Value
	}
}
