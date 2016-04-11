package dev

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/limbo-services/trace"
)

func NewHandler(l *log.Logger) trace.Handler {
	if l == nil {
		l = log.New(os.Stderr, "", 0)
	}
	return &logHandler{Log: l}
}

type logHandler struct {
	Log *log.Logger
}

func (h *logHandler) HandleTrace(t *trace.Trace) error {
	var buf bytes.Buffer
	var keys = make([]string, 0, 16)

	fmt.Fprintf(&buf, "Trace: %s\n", t.ID)

	for _, span := range t.Spans {
		fmt.Fprintf(&buf, "  %d %q\n", span.SpanID, span.Metadata["trace/name"])

		tabw := tabwriter.NewWriter(&buf, 8, 8, 2, ' ', 0)
		if span.ParentSpanID > 0 {
			fmt.Fprintf(tabw, "   - parent:\t%d\n", span.ParentSpanID)
		}
		fmt.Fprintf(tabw, "   - duration:\t%s\n", time.Duration(span.End-span.Begin))

		keys = keys[:0]
		for k := range span.Metadata {
			if k == "trace/name" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(tabw, "   - %s:\t%q\n", k, span.Metadata[k])
		}
		tabw.Flush()

		for _, e := range span.LogEntries {
			fmt.Fprintf(&buf, "   %s\n", strings.Replace(e.Text, "\n", "\n   ", -1))
			if e.Error && len(e.Stack) > 0 {
				fmt.Fprintf(&buf, "   │ %s\n", strings.Replace(strings.TrimSpace(e.Stack), "\n", "\n   │ ", -1))
			}
		}
	}

	h.Log.Print(buf.String())
	return nil
}
