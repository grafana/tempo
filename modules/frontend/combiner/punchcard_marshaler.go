package combiner

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/grafana/tempo/pkg/tempopb"
	commonv1 "github.com/grafana/tempo/pkg/tempopb/common/v1"
	tracev1 "github.com/grafana/tempo/pkg/tempopb/trace/v1"
	"github.com/grafana/tempo/pkg/util"
)

type punchCardMarshaler struct{}

// IBM 029 keypunch encoding: maps ASCII characters to rows that get punched.
// Rows are numbered 12, 11, 0-9 from top to bottom.
// A punch is represented as 'O', no punch as ' '.
var punchCardEncoding = map[byte][]int{
	' ':  {},
	'A':  {12, 1},
	'B':  {12, 2},
	'C':  {12, 3},
	'D':  {12, 4},
	'E':  {12, 5},
	'F':  {12, 6},
	'G':  {12, 7},
	'H':  {12, 8},
	'I':  {12, 9},
	'J':  {11, 1},
	'K':  {11, 2},
	'L':  {11, 3},
	'M':  {11, 4},
	'N':  {11, 5},
	'O':  {11, 6},
	'P':  {11, 7},
	'Q':  {11, 8},
	'R':  {11, 9},
	'S':  {0, 2},
	'T':  {0, 3},
	'U':  {0, 4},
	'V':  {0, 5},
	'W':  {0, 6},
	'X':  {0, 7},
	'Y':  {0, 8},
	'Z':  {0, 9},
	'0':  {0},
	'1':  {1},
	'2':  {2},
	'3':  {3},
	'4':  {4},
	'5':  {5},
	'6':  {6},
	'7':  {7},
	'8':  {8},
	'9':  {9},
	'.':  {12, 3, 8},
	'<':  {12, 4, 8},
	'(':  {12, 5, 8},
	'+':  {12, 6, 8},
	'|':  {12, 7, 8},
	'&':  {12},
	'-':  {11},
	'/':  {0, 1},
	',':  {0, 3, 8},
	'%':  {0, 4, 8},
	'_':  {0, 5, 8},
	'>':  {0, 6, 8},
	'?':  {0, 7, 8},
	':':  {2, 8},
	'#':  {3, 8},
	'@':  {4, 8},
	'\'': {5, 8},
	'=':  {6, 8},
	'"':  {7, 8},
	'a':  {12, 1},
	'b':  {12, 2},
	'c':  {12, 3},
	'd':  {12, 4},
	'e':  {12, 5},
	'f':  {12, 6},
	'g':  {12, 7},
	'h':  {12, 8},
	'i':  {12, 9},
	'j':  {11, 1},
	'k':  {11, 2},
	'l':  {11, 3},
	'm':  {11, 4},
	'n':  {11, 5},
	'o':  {11, 6},
	'p':  {11, 7},
	'q':  {11, 8},
	'r':  {11, 9},
	's':  {0, 2},
	't':  {0, 3},
	'u':  {0, 4},
	'v':  {0, 5},
	'w':  {0, 6},
	'x':  {0, 7},
	'y':  {0, 8},
	'z':  {0, 9},
}

// rowLabels maps row indices to their display labels on the card
var rowLabels = [12]string{"12", "11", " 0", " 1", " 2", " 3", " 4", " 5", " 6", " 7", " 8", " 9"}

func (m *punchCardMarshaler) marshalToString(t proto.Message) (string, error) {
	switch v := t.(type) {
	case *tempopb.TraceByIDResponse:
		return traceToPunchCards(v.Trace)
	case *tempopb.Trace:
		return traceToPunchCards(v)
	}
	return "", util.ErrUnsupported
}

func traceToPunchCards(t *tempopb.Trace) (string, error) {
	if t == nil || len(t.ResourceSpans) == 0 {
		return renderPunchCard("EMPTY TRACE - NO DATA FOUND"), nil
	}

	var cards []string

	// Header card
	var traceID string
	if len(t.ResourceSpans) > 0 && len(t.ResourceSpans[0].ScopeSpans) > 0 && len(t.ResourceSpans[0].ScopeSpans[0].Spans) > 0 {
		traceID = hex.EncodeToString(t.ResourceSpans[0].ScopeSpans[0].Spans[0].TraceId)
	}
	cards = append(cards, renderPunchCard(fmt.Sprintf("TRACE %s", strings.ToUpper(traceID))))

	for _, rs := range t.ResourceSpans {
		serviceName := extractServiceName(rs.Resource.GetAttributes())
		if serviceName == "" {
			serviceName = "UNKNOWN"
		}

		for _, ss := range rs.ScopeSpans {
			for _, span := range ss.Spans {
				line := formatSpanForCard(serviceName, span)
				cards = append(cards, renderPunchCard(line))
			}
		}
	}

	// Footer card
	cards = append(cards, renderPunchCard("END OF TRACE - THANK YOU FOR USING GRAFANA TEMPO"))

	return strings.Join(cards, "\n"), nil
}

func formatSpanForCard(service string, span *tracev1.Span) string {
	spanID := hex.EncodeToString(span.SpanId)
	name := strings.ToUpper(span.Name)
	dur := time.Duration(span.EndTimeUnixNano-span.StartTimeUnixNano) * time.Nanosecond

	status := "OK"
	if span.Status != nil && span.Status.Code == tracev1.Status_STATUS_CODE_ERROR {
		status = "ERR"
	}

	// IBM punch cards are 80 columns wide
	line := fmt.Sprintf("%-12s %-16s %-20s %10s %3s", strings.ToUpper(service), spanID, name, dur.Round(time.Microsecond), status)
	return line
}

func extractServiceName(attrs []*commonv1.KeyValue) string {
	for _, attr := range attrs {
		if attr.Key == "service.name" {
			return attr.Value.GetStringValue()
		}
	}
	return ""
}

// rowIndexToCardRow converts a display row index (0-11) to the IBM card row number.
// Display rows: 0=row12, 1=row11, 2=row0, 3=row1, ..., 11=row9
func rowIndexToCardRow(row int) int {
	switch {
	case row == 0:
		return 12
	case row == 1:
		return 11
	default:
		return row - 2
	}
}

// renderPunchCard renders a string as an ASCII art IBM 80-column punch card.
func renderPunchCard(text string) string {
	// Pad or truncate to exactly 80 characters
	if len(text) > 80 {
		text = text[:80]
	}
	text = fmt.Sprintf("%-80s", text)

	var sb strings.Builder

	// Top border with printed text
	sb.WriteString(" _______________________________________________________________________________\n")
	fmt.Fprintf(&sb, "/%-80s|\n", text)

	// 12 rows of punches
	for row := range 12 {
		fmt.Fprintf(&sb, "| %s ", rowLabels[row])
		for col := range 80 {
			ch := text[col]
			punches, ok := punchCardEncoding[ch]
			if !ok {
				sb.WriteByte(' ')
				continue
			}

			punchRow := rowIndexToCardRow(row)

			if slices.Contains(punches, punchRow) {
				sb.WriteByte('O')
			} else {
				sb.WriteByte(' ')
			}
		}
		sb.WriteString(" |\n")
	}

	// Bottom border
	sb.WriteString("|________________________________________________________________________________|")

	return sb.String()
}
