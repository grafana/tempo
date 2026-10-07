package html

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/compare"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/internal/comparetest"
	"github.com/grafana/tempo/v3/pkg/benchmark/compare/render"
	"github.com/grafana/tempo/v3/pkg/benchmark/metrics"
)

func get(t *testing.T, h http.Handler, target string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	return rec.Code, string(body)
}

func newTestHandler(t *testing.T, c *compare.Comparison) http.Handler {
	t.Helper()
	h, err := Handler(c, []string{"harness.wallNs", "backend.reads"}, compare.P50)
	require.NoError(t, err)
	return h
}

func TestSummaryPage(t *testing.T) {
	h := newTestHandler(t, comparetest.Comparison(t))

	code, body := get(t, h, "/")
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, body, "<title>harness.wallNs · tempo-cli benchmark compare</title>")
	require.Contains(t, body, "runs in order of readBufferSize; baseline base")
	require.Contains(t, body, "<h2>harness.wallNs · p50 per execution · change from base</h2>")

	// The runs are listed with what they changed, each a link to making it the
	// baseline, in its colour from the palette.
	require.Contains(t, body, `<span class="">readBufferSize default → 4MiB</span>`)
	require.Contains(t, body, `<span class="dim">gitSHA abc</span>`)
	require.Contains(t, body, `<a href="/?baseline=3&amp;metric=harness.wallNs&amp;p=p50" style="color: #0b7fa3" class="">4MiB</a>`)
	require.Contains(t, body, `style="color: #a3329a" class="bold">base</a>`, "the baseline stands out")

	// The table has a value and a change for each run, styled by how it moved.
	require.Contains(t, body, `<tr class="group"><th class="dim" colspan="99">traceByID</th></tr>`)
	require.Contains(t, body, `<a href="/case?baseline=0&amp;id=traceid%2Fpresent&amp;metric=harness.wallNs&amp;p=p50">traceid/present</a>`)
	require.Contains(t, body, `<td class="sep">87ns</td><td class="better bold">-13.0%</td>`)
	require.Contains(t, body, `<td class="sep">104ns</td><td class="worse">&#43;4.0%</td>`, "a plus sign, as the template writes it")
	require.Contains(t, body, `<td class="sep">10ns</td><td class="dim">0%</td>`)
	require.Contains(t, body, `<td class="sep warn" colspan="2">not comparable</td>`)
	require.Contains(t, body, "<li>search/nopredicate vs 4MiB: matched 5 vs 6</li>")
	require.Contains(t, body, `<th class="sep bold" colspan="1" style="color: #a3329a">base</th>`, "the baseline's column is one wide")
	require.Contains(t, body, `<th class="sep" colspan="2" style="color: #2e7d4f">again</th>`)

	// The styles' colours come from the palette.
	require.Contains(t, body, ".better { color: #0969da; }")
	require.Contains(t, body, ".bold { font-weight: 600; }")

	// Tabs switch the metric and the percentile.
	require.Contains(t, body, `<a href="/?baseline=0&amp;metric=backend.reads&amp;p=p50" class="">backend.reads</a>`)
	require.Contains(t, body, `<a href="/?baseline=0&amp;metric=harness.wallNs&amp;p=p50" class="active">p50</a>`)
	require.NotContains(t, body, "ZgotmplZ", "every value made it through the template's escaping")
}

func TestSummaryPageTotals(t *testing.T) {
	withTotal := func(m metrics.Measurement, total float64) metrics.Measurement {
		m.Total = total
		return m
	}
	run := func(name string, searchExecutions int) compare.Run {
		r := comparetest.Result("mac",
			comparetest.Case("traceid/present", 200, metrics.Set{
				"harness.wallNs": withTotal(comparetest.Measurement(50), 5000),
			}),
			comparetest.Case("search/nopredicate", 5, metrics.Set{
				"harness.wallNs": withTotal(comparetest.Measurement(10), 5500),
			}),
		)
		r.Cases[0].API = "traceByID"
		// The search case's executions measure different amounts of work, as
		// when the runs' blocks shard differently, but the match counts agree.
		r.Cases[1].Executions = searchExecutions
		return compare.Run{Name: name, Result: r}
	}
	c, err := compare.New([]compare.Run{run("base", 55), run("200mb", 28)})
	require.NoError(t, err)
	h := newTestHandler(t, c)

	code, body := get(t, h, "/")
	require.Equal(t, http.StatusOK, code)

	// The page teaches why some per-execution comparisons are withheld.
	require.Contains(t, body, `<p class="note">Shard counts differ between runs`)

	// The per-execution table keeps the case whose shard counts agree, and
	// drops the one they do not; the totals table compares every case.
	perExec := strings.Index(body, "p50 per execution · change from base")
	totals := strings.Index(body, "case total per pass · change from base")
	require.NotEqual(t, -1, perExec)
	require.NotEqual(t, -1, totals)
	require.Contains(t, body[perExec:totals], "traceid/present")
	require.NotContains(t, body[perExec:totals], "search/nopredicate")
	require.Contains(t, body[totals:], "search/nopredicate")
	require.Contains(t, body[totals:], "traceid/present")
}

func TestSummaryPageView(t *testing.T) {
	h := newTestHandler(t, comparetest.Comparison(t))

	code, body := get(t, h, "/?metric=backend.reads&p=p90&baseline=3")
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, body, "<h2>backend.reads · p90 per execution · change from 4MiB</h2>")
	require.Contains(t, body, "runs in order of readBufferSize; baseline 4MiB")

	for target, want := range map[string]string{
		"/?metric=nope":   `unknown metric "nope"`,
		"/?p=p42":         `unknown percentile "p42"`,
		"/?baseline=9":    `no run "9" to be the baseline`,
		"/?baseline=base": `no run "base" to be the baseline`,
	} {
		code, body := get(t, h, target)
		require.Equal(t, http.StatusBadRequest, code, target)
		require.Contains(t, body, want, target)
	}

	code, _ = get(t, h, "/nowhere")
	require.Equal(t, http.StatusNotFound, code)
}

func TestCasePage(t *testing.T) {
	c := comparetest.Comparison(t)
	c.Cases[1].Query = "{}"
	h := newTestHandler(t, c)

	code, body := get(t, h, "/case?id=search/nopredicate&baseline=0")
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, body, "<h2>search/nopredicate · per execution</h2>")
	require.Contains(t, body, "query: <code>{}</code>")
	require.Contains(t, body, "<li>vs 4MiB: matched 5 vs 6</li>")
	require.Contains(t, body, `‹ previous case`)
	require.NotContains(t, body, `next case ›`, "the last case has none after it")

	// Every metric has its box plot and table; one the case does not report
	// still has a row per run.
	require.Equal(t, 2, strings.Count(body, `<svg class="boxplot"`))
	require.Contains(t, body, "<h2>harness.wallNs</h2>")
	require.Contains(t, body, "<h2>backend.reads</h2>")
	require.Contains(t, body, "<th>Δp50</th>")
	require.Contains(t, body, `<td style="color: #2e7d4f" class="">again</td><td>10</td><td>10ns</td>`)
	require.Contains(t, body, "no data")
	require.NotContains(t, body, "ZgotmplZ")

	// The baseline links on a case page stay on the case.
	require.Contains(t, body, `href="/case?baseline=2&amp;id=search%2Fnopredicate&amp;metric=harness.wallNs&amp;p=p50"`)

	code, _ = get(t, h, "/case?id=nope")
	require.Equal(t, http.StatusNotFound, code)
	code, _ = get(t, h, "/case?id=search/nopredicate&p=p42")
	require.Equal(t, http.StatusBadRequest, code)
}

func TestPagesEscape(t *testing.T) {
	r := func(name string) compare.Run {
		return compare.Run{Name: name, Result: comparetest.Result("mac", comparetest.Case("<script>case", 1, metrics.Set{"harness.wallNs": comparetest.Measurement(10)}))}
	}
	c, err := compare.New([]compare.Run{r("<b>base</b>"), r("<i>next")})
	require.NoError(t, err)
	h := newTestHandler(t, c)

	for _, target := range []string{"/", "/case?id=%3Cscript%3Ecase"} {
		code, body := get(t, h, target)
		require.Equal(t, http.StatusOK, code, target)
		require.NotContains(t, body, "<b>base</b>", target)
		require.NotContains(t, body, "<i>next", target)
		require.NotContains(t, body, "<script>", target)
		require.Contains(t, body, "&lt;b&gt;base&lt;/b&gt;", target)
	}
}

func TestBoxPlotSVG(t *testing.T) {
	far := comparetest.Summary(15, 26, 32, 37, 40, 48, 1116)
	near := comparetest.Summary(14, 23, 28, 33, 36, 44, 49)
	s := compare.Series{Unit: compare.Count, Summaries: []*metrics.Summary{&far, &near, nil}}
	svg := string(boxPlotSVG(newBoxPlot([]string{"base", "a&b", "gone"}, s, 0)))

	require.True(t, strings.HasPrefix(svg, `<svg class="boxplot"`))
	require.Equal(t, 2, strings.Count(svg, "<rect "), "a box per run with data")
	require.Contains(t, svg, `fill="#a3329a" font-weight="bold">base</text>`, "the baseline is in bold, in its colour")
	require.Contains(t, svg, ">a&amp;b</text>", "names are escaped")
	require.Contains(t, svg, "<title>base: min 15 · p25 26 · p50 32 · p75 37 · p90 40 · p99 48 · max 1.12k</title>")
	require.Contains(t, svg, `<text class="clipped"`, "a max past the axis is written out at its edge")
	require.Equal(t, 1, strings.Count(svg, "<circle "), "a max on the axis is marked on it")
	require.Contains(t, svg, ">no data</text>")
	require.Contains(t, svg, `text-anchor="start">10</text>`, "the axis starts at a round number")
	require.Contains(t, svg, `text-anchor="end">50</text>`)
}

func TestClass(t *testing.T) {
	require.Equal(t, "", class(render.Plain))
	require.Equal(t, "dim", class(render.Dim))
	require.Equal(t, "better bold", class(render.MuchBetter))
	require.Equal(t, "worse", class(render.Worse))
	require.Equal(t, "bold", class(render.BaselineName))
	require.Equal(t, "bold", class(render.Title))
}
