package trace

import "sync"

var tracePool sync.Pool
var spanPool sync.Pool

func (t *Trace) release() {
	if t == nil {
		return
	}

	for i, span := range t.Spans {
		t.Spans[i] = nil
		span.release()
	}

	blank := Trace{
		Spans: t.Spans[:0],
	}
	*t = blank

	tracePool.Put(t)
}

func getTrace() *Trace {
	t, _ := tracePool.Get().(*Trace)
	if t == nil {
		t = &Trace{
			Spans: make([]*Span, 0, 16),
		}
	}
	return t
}

func getSpan() *Span {
	s, _ := spanPool.Get().(*Span)
	if s == nil {
		s = &Span{
			LogEntries: make([]LogEntry, 0, 16),
			Metadata:   make(map[string]string, 16),
		}
	}
	return s
}

func (s *Span) release() {
	if s == nil {
		return
	}

	for i := range s.LogEntries {
		s.LogEntries[i] = LogEntry{}
	}

	if s.Metadata != nil {
		for k := range s.Metadata {
			delete(s.Metadata, k)
		}
	}

	blank := Span{
		LogEntries: s.LogEntries[:0],
		Metadata:   s.Metadata,
	}
	*s = blank

	spanPool.Put(s)
}
