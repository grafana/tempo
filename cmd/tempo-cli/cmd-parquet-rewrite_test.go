package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/tempopb"
	"github.com/grafana/tempo/v3/pkg/tracefingerprint"
	"github.com/grafana/tempo/v3/pkg/util/test"
	"github.com/grafana/tempo/v3/tempodb/backend"
	"github.com/grafana/tempo/v3/tempodb/backend/local"
	"github.com/grafana/tempo/v3/tempodb/encoding"
	"github.com/grafana/tempo/v3/tempodb/encoding/common"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet4"
	"github.com/grafana/tempo/v3/tempodb/encoding/vparquet5"
)

// writeParquetRewriteBloom writes a bloom filter created with the given parameters
// as bloom-N files in dir, matching how block creation writes bloom shards.
func writeParquetRewriteBloom(t *testing.T, dir string, fp float64, shardSizeBytes int, n int) (bitsPerShard, k uint64, shardCount int) {
	t.Helper()

	b := common.NewBloom(fp, uint(shardSizeBytes), uint(n))
	shards, err := b.Marshal()
	require.NoError(t, err)
	for i, shard := range shards {
		require.NoError(t, os.WriteFile(filepath.Join(dir, common.BloomName(i)), shard, 0o600))
	}

	var header [16]byte
	copy(header[:], shards[0][:16])
	bitsPerShard = binary.BigEndian.Uint64(header[0:8])
	k = binary.BigEndian.Uint64(header[8:16])

	return bitsPerShard, k, b.GetShardCount()
}

func TestParquetIteratorsYieldAllTraces(t *testing.T) {
	ctx := context.Background()

	t.Run("vparquet4", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "data.parquet")
		f, err := os.Create(path)
		require.NoError(t, err)

		w := parquet.NewGenericWriter[*vparquet4.Trace](f)
		traces := make([]*vparquet4.Trace, 0, 3)
		for i := range 3 {
			traces = append(traces, &vparquet4.Trace{
				TraceID:      []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(i + 1)},
				TraceIDText:  fmt.Sprintf("%032d", i+1),
				RootSpanName: fmt.Sprintf("root-%d", i),
			})
		}
		_, err = w.Write(traces)
		require.NoError(t, err)
		require.NoError(t, w.Close())
		require.NoError(t, f.Close())

		in, err := os.Open(path)
		require.NoError(t, err)
		t.Cleanup(func() { _ = in.Close() })
		stat, err := in.Stat()
		require.NoError(t, err)
		pf, err := parquet.OpenFile(in, stat.Size())
		require.NoError(t, err)

		iter := &parquetIterator4{
			r: parquet.NewGenericReader[*vparquet4.Trace](pf),
			m: &backend.BlockMeta{},
		}
		t.Cleanup(iter.Close)

		yielded := 0
		for {
			_, tr, err := iter.Next(ctx)
			if tr != nil {
				yielded++
			}
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
		}
		require.Equal(t, 3, yielded)
	})
}

func TestInheritedBloomParams(t *testing.T) {
	// inheritedBloomParams must reconstruct parameters that reproduce the original
	// bloom: same shard size, same hash count k, and same shard count.
	tests := []struct {
		name           string
		fp             float64
		shardSizeBytes int
		n              int
	}{
		// common.NewBloom(.01, 100*1024, n) creates k=7 shards of 100KiB
		{"default-like bloom", 0.01, 100 * 1024, 1_000_000},
		{"small block", 0.01, 100 * 1024, 100},
		{"tiny shard", 0.01, 1024, 50},
		{"high fp", 0.99, 1024 * 1024, 162},
		{"multi shard", 0.01, 1024, 1_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			bitsPerShard, k, shardCount := writeParquetRewriteBloom(t, dir, tt.fp, tt.shardSizeBytes, tt.n)
			meta := &backend.BlockMeta{
				TotalObjects:    int64(tt.n),
				BloomShardCount: uint32(shardCount),
			}

			inheritedFP, shardSizeBytes, err := inheritedBloomParams(dir, meta)
			require.NoError(t, err)
			require.Equal(t, tt.shardSizeBytes, shardSizeBytes)
			require.Greater(t, inheritedFP, 0.0)
			require.Less(t, inheritedFP, 1.0)

			// the reconstructed rate must reproduce the original bloom
			rebuilt := common.NewBloom(inheritedFP, uint(shardSizeBytes), uint(tt.n))
			require.Equal(t, shardCount, rebuilt.GetShardCount())

			rebuiltShards, err := rebuilt.Marshal()
			require.NoError(t, err)
			var header [16]byte
			copy(header[:], rebuiltShards[0][:16])
			require.Equal(t, bitsPerShard, binary.BigEndian.Uint64(header[0:8]))
			require.Equal(t, k, binary.BigEndian.Uint64(header[8:16]))
		})
	}

	t.Run("missing bloom shard", func(t *testing.T) {
		_, _, err := inheritedBloomParams(t.TempDir(), &backend.BlockMeta{})
		require.Error(t, err)
	})

	t.Run("truncated header", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, common.BloomName(0)), make([]byte, 8), 0o600))

		_, _, err := inheritedBloomParams(dir, &backend.BlockMeta{})
		require.Error(t, err)
	})

	t.Run("invalid header", func(t *testing.T) {
		tests := []struct {
			name string
			m, k uint64
		}{
			{"zero bits", 0, 7},
			{"non-byte-aligned bits", 8193, 7},
			{"zero hash functions", 8192, 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := t.TempDir()
				var header bytes.Buffer
				require.NoError(t, binary.Write(&header, binary.BigEndian, tt.m))
				require.NoError(t, binary.Write(&header, binary.BigEndian, tt.k))
				require.NoError(t, os.WriteFile(filepath.Join(dir, common.BloomName(0)), header.Bytes(), 0o600))

				_, _, err := inheritedBloomParams(dir, &backend.BlockMeta{})
				require.Error(t, err)
			})
		}
	})
}

func TestParquetRewriteBlockConfig(t *testing.T) {
	const (
		fixtureShardSizeBytes = 10240
		fixtureObjects        = 1000
	)

	newFixture := func(t *testing.T) (string, *backend.BlockMeta, float64) {
		t.Helper()

		dir := t.TempDir()
		bitsPerShard, k, shardCount := writeParquetRewriteBloom(t, dir, 0.03, fixtureShardSizeBytes, fixtureObjects)
		meta := &backend.BlockMeta{
			TotalObjects:    fixtureObjects,
			BloomShardCount: uint32(shardCount),
		}
		// the rate blockConfig inherits for this fixture. TestInheritedBloomParams
		// pins that this rate reproduces the fixture's bloom.
		inheritedFP := reconstructBloomFP(fixtureObjects, bitsPerShard, uint64(shardCount), k)

		return dir, meta, inheritedFP
	}

	tests := []struct {
		name    string
		cmd     parquetRewrite
		want    common.BlockConfig
		wantErr bool
	}{
		{
			name: "row group size defaults, bloom inherited",
			cmd:  parquetRewrite{},
			want: common.BlockConfig{
				RowGroupSizeBytes:   defaultRowGroupSizeBytes,
				BloomFP:             0, // filled in below from the fixture
				BloomShardSizeBytes: fixtureShardSizeBytes,
			},
		},
		{
			name: "all parameters overridden",
			cmd: parquetRewrite{
				RowGroupSizeBytes:   555,
				BloomFP:             0.02,
				BloomShardSizeBytes: 999,
			},
			want: common.BlockConfig{
				RowGroupSizeBytes:   555,
				BloomFP:             0.02,
				BloomShardSizeBytes: 999,
			},
		},
		{
			name: "row group size overridden, bloom inherited",
			cmd:  parquetRewrite{RowGroupSizeBytes: 555},
			want: common.BlockConfig{
				RowGroupSizeBytes:   555,
				BloomFP:             0, // filled in below from the fixture
				BloomShardSizeBytes: fixtureShardSizeBytes,
			},
		},
		{
			name: "bloom fp overridden, row group size defaults",
			cmd:  parquetRewrite{BloomFP: 0.02},
			want: common.BlockConfig{
				RowGroupSizeBytes:   defaultRowGroupSizeBytes,
				BloomFP:             0.02,
				BloomShardSizeBytes: fixtureShardSizeBytes,
			},
		},
		{
			name:    "negative row group size",
			cmd:     parquetRewrite{RowGroupSizeBytes: -1},
			wantErr: true,
		},
		{
			name:    "bloom fp out of range",
			cmd:     parquetRewrite{BloomFP: 3},
			wantErr: true,
		},
		{
			name:    "negative bloom shard size",
			cmd:     parquetRewrite{BloomShardSizeBytes: -1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, meta, inheritedFP := newFixture(t)
			tt.cmd.In = dir
			tt.want.Version = vparquet5.VersionString
			if tt.want.BloomFP == 0 && tt.cmd.BloomFP == 0 {
				tt.want.BloomFP = inheritedFP
			}

			got, err := tt.cmd.blockConfig(vparquet5.VersionString, meta)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, &tt.want, got)
		})
	}

	t.Run("missing bloom shard cannot be inherited", func(t *testing.T) {
		dir, meta, _ := newFixture(t)
		require.NoError(t, os.Remove(filepath.Join(dir, common.BloomName(0))))

		cmd := parquetRewrite{In: dir}
		_, err := cmd.blockConfig(vparquet5.VersionString, meta)
		require.Error(t, err)
	})
}

func TestParquetRewriteOutputOptions(t *testing.T) {
	const key = vparquet5.TraceSignatureVersionOption
	for _, tc := range []struct {
		name        string
		format      string
		options     map[string]string
		fingerprint string
		want        map[string]string
		wantErr     string
	}{
		{name: "unsigned stays unsigned", format: vparquet5.VersionString, want: nil},
		{name: "an explicit empty version is kept", format: vparquet5.VersionString, options: map[string]string{key: "", "other": "x"}, want: map[string]string{key: "", "other": "x"}},
		{name: "none keeps unsigned input unsigned", format: vparquet5.VersionString, fingerprint: "none", want: nil},
		{name: "none drops an explicit empty version", format: vparquet5.VersionString, options: map[string]string{key: "", "other": "x"}, fingerprint: "none", want: map[string]string{"other": "x"}},
		{name: "signed input needs a choice", format: vparquet5.VersionString, options: map[string]string{key: "v1"}, wantErr: "pass --fingerprint=v1 to recompute them or --fingerprint=none to drop them"},
		{name: "unknown input version needs a choice", format: vparquet5.VersionString, options: map[string]string{key: "v9"}, wantErr: "(version v9)"},
		{name: "none drops fingerprints", format: vparquet5.VersionString, options: map[string]string{key: "v1", "other": "x"}, fingerprint: "none", want: map[string]string{"other": "x"}},
		{name: "none drops an unknown version", format: vparquet5.VersionString, options: map[string]string{key: "v9"}, fingerprint: "none", want: map[string]string{}},
		{name: "fingerprint unsigned input", format: vparquet5.VersionString, fingerprint: "v1", want: map[string]string{key: "v1"}},
		{name: "fingerprint signed input", format: vparquet5.VersionString, options: map[string]string{key: "v1", "other": "x"}, fingerprint: "v1", want: map[string]string{key: "v1", "other": "x"}},
		{name: "vParquet4 cannot store fingerprints", format: vparquet4.VersionString, fingerprint: "v1", wantErr: "trace fingerprints need a vParquet5 block"},
		{name: "unsupported version", format: vparquet5.VersionString, fingerprint: "v2", wantErr: `unsupported --fingerprint "v2"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meta := &backend.BlockMeta{Version: tc.format, Options: maps.Clone(tc.options)}
			cmd := parquetRewrite{Fingerprint: tc.fingerprint}
			got, err := cmd.outputOptions(meta)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.options, meta.Options, "input meta must not change")
		})
	}
}

func TestParquetRewriteCheckReport(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name, format, report, wantErr string
	}{
		{name: "no report", format: vparquet4.VersionString},
		{name: "new file", format: vparquet5.VersionString, report: filepath.Join(dir, "report.json")},
		{name: "vParquet4", format: vparquet4.VersionString, report: filepath.Join(dir, "report.json"), wantErr: "--fingerprint-report needs a vParquet5 block"},
		{name: "deletion marker", format: vparquet5.VersionString, report: filepath.Join(dir, backend.DeletedMetaName), wantErr: "cannot be named meta.deleted.json"},
		{name: "nocompact flag in other case", format: vparquet5.VersionString, report: filepath.Join(dir, "NoCompact.FLG"), wantErr: "cannot be named"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parquetRewrite{FingerprintReport: tc.report}
			err := cmd.checkReport(&backend.BlockMeta{Version: tc.format})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestParquetRewriteFingerprints rewrites a block with fingerprints, then
// rewrites the result without them, re-signs a block whose stored
// fingerprints are wrong, and checks the reports.
func TestParquetRewriteFingerprints(t *testing.T) {
	traces := []*tempopb.Trace{{}} // no spans, so StatusEmpty
	for i := range 30 {
		traces = append(traces, test.MakeTrace(1+i%3, nil))
	}
	computed := len(traces) - 1
	input := writeRewriteInputBlock(t, traces)

	rewrite := func(t *testing.T, in, fingerprint string) (string, *backend.BlockMeta, fingerprintReport) {
		t.Helper()
		out := t.TempDir()
		reportPath := filepath.Join(out, "report.json")
		cmd := parquetRewrite{In: in, Out: out, RowGroupSizeBytes: 1 << 20, Fingerprint: fingerprint, FingerprintReport: reportPath}
		require.NoError(t, cmd.Run())

		inMeta, err := readBlockMeta(in)
		require.NoError(t, err)
		dir := filepath.Join(out, inMeta.TenantID, inMeta.BlockID.String())
		meta, err := readBlockMeta(dir)
		require.NoError(t, err)
		data, err := os.ReadFile(reportPath)
		require.NoError(t, err)
		var report fingerprintReport
		require.NoError(t, json.Unmarshal(data, &report))
		return dir, meta, report
	}
	checkMeasured := func(t *testing.T, report fingerprintReport, stored int) {
		t.Helper()
		require.Empty(t, report.FailedPhase)
		require.Empty(t, report.Error)
		require.EqualValues(t, len(traces), report.InputRows)
		require.EqualValues(t, len(traces), report.InputRowsRead)
		require.Zero(t, report.InputRowsFailed)
		require.Zero(t, report.InputRowsUnread)
		require.Positive(t, report.RewriteAllocBytes)
		require.EqualValues(t, len(traces), report.Rows)
		require.Equal(t, len(traces), report.Traces)
		require.Equal(t, computed, report.Computed)
		require.Equal(t, 1, report.Empty)
		require.Zero(t, report.Unavailable)
		require.Equal(t, stored, report.StoredFingerprints)
		require.Zero(t, report.StoredMismatches)
		require.Greater(t, report.BlockBytes, report.DataFileBytes)
		require.Equal(t, 1, report.Computation.AllocationSamples, "a small block still samples the first call")
		require.Equal(t, computed, report.Accuracy.SampledTraces)
		require.Equal(t, computed*(computed-1)/2, report.Accuracy.Pairs)
		require.EqualValues(t, len(traces), report.Shingles.Count, "the empty trace counts as 0 shingles")
		require.Zero(t, report.Shingles.Min)
		require.Positive(t, report.Shingles.Max)
	}

	signedDir, signedMeta, signed := rewrite(t, input, "v1")
	require.Equal(t, "v1", signedMeta.Options[vparquet5.TraceSignatureVersionOption])
	require.Equal(t, "v1", signed.FingerprintVersion)
	require.Positive(t, signed.SimilarityColumnBytes)
	require.FileExists(t, filepath.Join(signedDir, backend.NoCompactFileName))
	checkMeasured(t, signed, computed)

	cmd := parquetRewrite{In: signedDir, Out: t.TempDir()}
	require.ErrorContains(t, cmd.Run(), "--fingerprint=none", "a signed input is not stripped by default")

	// The default output root of a block rewritten earlier holds that block,
	// so a rewrite of it would write into its own input.
	signedData, err := os.ReadFile(filepath.Join(signedDir, vparquet5.DataFileName))
	require.NoError(t, err)
	cmd = parquetRewrite{In: signedDir, Out: filepath.Dir(filepath.Dir(signedDir)), RowGroupSizeBytes: 1 << 20, Fingerprint: "none"}
	require.ErrorContains(t, cmd.Run(), "already exists")
	after, err := os.ReadFile(filepath.Join(signedDir, vparquet5.DataFileName))
	require.NoError(t, err)
	require.Equal(t, signedData, after, "the input must be intact")

	unsignedDir, unsignedMeta, unsigned := rewrite(t, signedDir, "none")
	require.NotContains(t, unsignedMeta.Options, vparquet5.TraceSignatureVersionOption)
	require.Empty(t, unsigned.FingerprintVersion)
	require.Zero(t, unsigned.SimilarityColumnBytes)
	require.NoFileExists(t, filepath.Join(unsignedDir, backend.NoCompactFileName))
	checkMeasured(t, unsigned, 0)
	require.Equal(t, signed.Shingles, unsigned.Shingles)

	// The report must see every wrong stored value, including one where the
	// trace has no fingerprint, and fail. A rewrite must replace them.
	stale, _, _ := rewrite(t, input, "v1")
	corruptFingerprints(t, stale)
	staleMeta, err := readBlockMeta(stale)
	require.NoError(t, err)
	staleReport := newFingerprintReport(staleMeta.BlockID.String(), rewriteStats{})
	require.ErrorContains(t, staleReport.measure(stale, staleMeta), "differ from the traces read back")
	require.Equal(t, phaseReadback, staleReport.FailedPhase)
	require.Equal(t, len(traces), staleReport.Traces, "every row is still measured")
	require.Equal(t, len(traces), staleReport.StoredFingerprints)
	require.Equal(t, len(traces), staleReport.StoredMismatches)
	_, _, resigned := rewrite(t, stale, "v1")
	checkMeasured(t, resigned, computed)
}

// TestParquetRewriteReportPath checks that a report path that could replace or
// mislead block files fails before the rewrite and leaves no output behind.
func TestParquetRewriteReportPath(t *testing.T) {
	input := writeRewriteInputBlock(t, []*tempopb.Trace{test.MakeTrace(2, nil)})
	inMeta, err := readBlockMeta(input)
	require.NoError(t, err)
	metaPath := filepath.Join(input, backend.MetaName)
	inputMeta, err := os.ReadFile(metaPath)
	require.NoError(t, err)
	existing := filepath.Join(t.TempDir(), "report.json")
	require.NoError(t, os.WriteFile(existing, []byte("keep"), 0o600))

	out := t.TempDir()
	outDir := filepath.Join(out, inMeta.TenantID, inMeta.BlockID.String())
	for _, tc := range []struct {
		name, report, wantErr string
		wantIs                error
	}{
		{name: "input meta", report: metaPath, wantErr: "cannot be named meta.json"},
		{name: "new file in the input block", report: filepath.Join(input, "report.json"), wantErr: "is in block directory"},
		{name: "output data file", report: filepath.Join(outDir, vparquet5.DataFileName), wantErr: "is in block directory"},
		{name: "existing file", report: existing, wantIs: fs.ErrExist},
		{name: "missing directory", report: filepath.Join(out, "missing", "report.json"), wantIs: fs.ErrNotExist},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := parquetRewrite{In: input, Out: out, RowGroupSizeBytes: 1 << 20, FingerprintReport: tc.report}
			err := cmd.Run()
			if tc.wantIs != nil {
				require.ErrorIs(t, err, tc.wantIs)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			require.NoDirExists(t, outDir)
		})
	}
	require.NoFileExists(t, filepath.Join(input, "report.json"))
	after, err := os.ReadFile(metaPath)
	require.NoError(t, err)
	require.Equal(t, inputMeta, after)
	after, err = os.ReadFile(existing)
	require.NoError(t, err)
	require.Equal(t, "keep", string(after))
}

// TestParquetRewriteReportsRewriteFailure checks that a rewrite that fails
// partway writes a report saying which rows were read, removes its partial
// output, and can then be retried.
func TestParquetRewriteReportsRewriteFailure(t *testing.T) {
	input := writeRewriteInputBlock(t, []*tempopb.Trace{test.MakeTrace(2, nil)})
	meta, err := readBlockMeta(input)
	require.NoError(t, err)
	enc, err := encoding.FromVersionForWrites(meta.Version)
	require.NoError(t, err)
	out := t.TempDir()
	blockDir := filepath.Join(out, meta.TenantID, meta.BlockID.String())

	for _, tc := range []struct {
		name           string
		err            error
		wantErr        string
		failed, unread int64
	}{
		{name: "row read error", err: errors.New("bad row"), wantErr: "bad row", failed: 1, unread: 2},
		{name: "input ends early", err: io.EOF, wantErr: "the input ended after 2 of 5 rows", unread: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reportPath := filepath.Join(t.TempDir(), "report.json")
			cmd := parquetRewrite{In: input, Out: out, RowGroupSizeBytes: 1 << 20, Fingerprint: "v1", FingerprintReport: reportPath}
			options, err := cmd.outputOptions(meta)
			require.NoError(t, err)
			iter := &failingIterator{testIterator: testIterator{traces: newTestTraces(2)}, err: tc.err}
			require.ErrorContains(t, cmd.writeBlock(enc, meta, options, 5, iter), tc.wantErr)
			require.NoDirExists(t, blockDir, "the partial output is removed")

			data, err := os.ReadFile(reportPath)
			require.NoError(t, err)
			var report fingerprintReport
			require.NoError(t, json.Unmarshal(data, &report))
			require.Equal(t, phaseRewrite, report.FailedPhase)
			require.Contains(t, report.Error, tc.wantErr)
			require.EqualValues(t, 5, report.InputRows)
			require.EqualValues(t, 2, report.InputRowsRead)
			require.Equal(t, tc.failed, report.InputRowsFailed)
			require.Equal(t, tc.unread, report.InputRowsUnread)
			require.Zero(t, report.Traces)
		})
	}

	cmd := parquetRewrite{In: input, Out: out, RowGroupSizeBytes: 1 << 20, Fingerprint: "v1"}
	options, err := cmd.outputOptions(meta)
	require.NoError(t, err)
	require.NoError(t, cmd.writeBlock(enc, meta, options, 2, &testIterator{traces: newTestTraces(2)}), "retry after a failure")
	require.DirExists(t, blockDir)
}

// TestParquetRewriteInputColumns checks the input's columns against its meta.
// Columns the meta describes but the file lacks are refused before reading.
// Fingerprint columns of a version this code does not know are not read, so
// such a block can be rewritten with either choice the error suggests even if
// those columns do not decode.
func TestParquetRewriteInputColumns(t *testing.T) {
	relabel := func(t *testing.T, dir, version string) {
		t.Helper()
		meta, err := readBlockMeta(dir)
		require.NoError(t, err)
		meta.Options = map[string]string{vparquet5.TraceSignatureVersionOption: version}
		data, err := json.Marshal(meta)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, backend.MetaName), data, 0o600))
	}
	rewrite := func(t *testing.T, in, fingerprint string) (string, fingerprintReport) {
		t.Helper()
		out := t.TempDir()
		reportPath := filepath.Join(out, "report.json")
		cmd := parquetRewrite{In: in, Out: out, RowGroupSizeBytes: 1 << 20, Fingerprint: fingerprint, FingerprintReport: reportPath}
		require.NoError(t, cmd.Run(), "--fingerprint=%s", fingerprint)
		data, err := os.ReadFile(reportPath)
		require.NoError(t, err)
		var report fingerprintReport
		require.NoError(t, json.Unmarshal(data, &report))
		meta, err := readBlockMeta(in)
		require.NoError(t, err)
		return filepath.Join(out, meta.TenantID, meta.BlockID.String()), report
	}

	unsigned := writeRewriteInputBlock(t, []*tempopb.Trace{test.MakeTrace(2, nil)})
	relabel(t, unsigned, vparquet5.TraceSignatureVersionV1)
	cmd := parquetRewrite{In: unsigned, Out: t.TempDir(), RowGroupSizeBytes: 1 << 20, Fingerprint: "none"}
	require.ErrorContains(t, cmd.Run(), "but the block meta describes")

	input := writeRewriteInputBlock(t, []*tempopb.Trace{test.MakeTrace(2, nil), test.MakeTrace(3, nil)})
	signed, want := rewrite(t, input, "v1")
	relabel(t, signed, vparquet5.TraceSignatureVersionNone)
	cmd = parquetRewrite{In: signed, Out: t.TempDir(), RowGroupSizeBytes: 1 << 20}
	require.ErrorContains(t, cmd.Run(), "but the block meta describes", "fingerprints the meta omits are not dropped silently")

	relabel(t, signed, "v9")
	corruptSimilarityPages(t, signed)
	for _, fingerprint := range []string{"none", "v1"} {
		_, got := rewrite(t, signed, fingerprint)
		require.Equal(t, 2, got.Traces)
		require.Zero(t, got.StoredMismatches)
		require.Equal(t, want.Shingles, got.Shingles, "the traces survive the rewrite")
	}
}

// corruptSimilarityPages overwrites the Similarity column chunks of the
// single-row-group block in dir with bytes that do not decode, leaving the
// other columns intact.
func corruptSimilarityPages(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, vparquet5.DataFileName)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	pf, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	require.Len(t, pf.Metadata().RowGroups, 1)
	corrupted := 0
	for _, col := range pf.Metadata().RowGroups[0].Columns {
		if !isSimilarityColumn(col.MetaData.PathInSchema) {
			continue
		}
		start := col.MetaData.DataPageOffset
		if d := col.MetaData.DictionaryPageOffset; d > 0 && d < start {
			start = d
		}
		for i := start; i < start+col.MetaData.TotalCompressedSize; i++ {
			data[i] = 0xff
		}
		corrupted++
	}
	require.Positive(t, corrupted)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

type failingIterator struct {
	testIterator
	err error
}

func (i *failingIterator) Next(ctx context.Context) (common.ID, *tempopb.Trace, error) {
	id, tr, err := i.testIterator.Next(ctx)
	if errors.Is(err, io.EOF) {
		return nil, nil, i.err
	}
	return id, tr, err
}

func TestMeasureAccuracy(t *testing.T) {
	trace := func(shingles []uint64, onesBytes ...int) sampledTrace {
		res := tracefingerprint.Result{Version: tracefingerprint.Version, Status: tracefingerprint.StatusComputed, ShingleCount: uint32(len(shingles))}
		for _, i := range onesBytes {
			res.Fingerprint[i] = 0xff
		}
		return sampledTrace{result: res, shingles: shingles}
	}
	// a and b each differ from q in 64 of 512 bits, so both are estimated at
	// 0.75 against q, but their exact similarities to q are 0.8 and 0.5. They
	// differ from each other in 128 bits, an estimate of 0.5 against an exact
	// 4/6.
	q := trace([]uint64{1, 2, 3, 4})
	a := trace([]uint64{1, 2, 3, 4, 5}, 0, 1, 2, 3, 4, 5, 6, 7)
	b := trace([]uint64{1, 2, 3, 5, 6}, 8, 9, 10, 11, 12, 13, 14, 15)

	acc := measureAccuracy([]sampledTrace{q, a, b})
	require.Equal(t, 3, acc.SampledTraces)
	require.Equal(t, 3, acc.Pairs)
	require.InDelta(t, (0.05+0.25+1.0/6)/3, acc.MeanAbsError, 1e-9)
	require.InDelta(t, 0.25, acc.P99AbsError, 1e-9)
	require.InDelta(t, 0.25, acc.MaxAbsError, 1e-9)
	require.Equal(t, map[string]int{"0.5": 0, "0.8": 1, "0.9": 0}, acc.ThresholdFlips)
	require.Zero(t, acc.FalseExactMatches)
	require.Equal(t, ties{Comparisons: 3, Tied: 1, TiedDifferentExact: 1, TiedDifferentExactAboveZero: 1}, acc.Ties)
}

// writeRewriteInputBlock writes traces as an unsigned vParquet5 block and
// returns the block directory.
func writeRewriteInputBlock(t *testing.T, traces []*tempopb.Trace) string {
	t.Helper()
	root := t.TempDir()
	rawR, rawW, _, err := local.New(&local.Config{Path: root})
	require.NoError(t, err)

	sorted := make([]testTrace, 0, len(traces))
	for _, tr := range traces {
		sorted = append(sorted, testTrace{traceID: test.ValidTraceID(nil), trace: tr})
	}
	slices.SortFunc(sorted, func(a, b testTrace) int { return bytes.Compare(a.traceID, b.traceID) })

	meta := backend.NewBlockMeta("tenant", uuid.New(), vparquet5.VersionString)
	meta.TotalObjects = int64(len(traces))
	cfg := &common.BlockConfig{BloomFP: 0.01, BloomShardSizeBytes: 100 * 1024, RowGroupSizeBytes: 1 << 20}
	_, err = vparquet5.CreateBlock(context.Background(), cfg, meta, &testIterator{traces: sorted}, backend.NewReader(rawR), backend.NewWriter(rawW))
	require.NoError(t, err)
	return filepath.Join(root, meta.TenantID, meta.BlockID.String())
}

// corruptFingerprints rewrites the data file of the signed block in dir so
// that every row stores a wrong, non-zero fingerprint and shingle count.
func corruptFingerprints(t *testing.T, dir string) {
	t.Helper()
	meta, err := readBlockMeta(dir)
	require.NoError(t, err)
	path := filepath.Join(dir, vparquet5.DataFileName)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	schema, writerOptions, readerOptions := vparquet5.SchemaWithDynamicChanges(vparquet5.TraceSchemaConfig{DedicatedColumns: meta.DedicatedColumns, Options: meta.Options})
	pf, err := parquet.OpenFile(bytes.NewReader(data), int64(len(data)), parquet.FileSchema(schema))
	require.NoError(t, err)

	r := parquet.NewGenericReader[*vparquet5.Trace](pf, readerOptions...)
	var rows []*vparquet5.Trace
	for {
		batch := []*vparquet5.Trace{{}}
		n, err := r.Read(batch)
		rows = append(rows, batch[:n]...)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	require.NoError(t, r.Close())
	require.EqualValues(t, pf.NumRows(), len(rows))
	for _, row := range rows {
		row.Similarity.Signature[0] ^= 1
		row.Similarity.ShingleCount++
	}

	var buf bytes.Buffer
	w := parquet.NewGenericWriter[*vparquet5.Trace](&buf, writerOptions...)
	_, err = w.Write(rows)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o600))
}
