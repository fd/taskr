package trace

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"
)

type Trace struct {
	ID    string  `json:"i,omitempty"`
	Spans []*Span `json:"s,omitempty"`

	nextSpanID uint64 // using atomic
	tracer     *Tracer
	root       *Span

	mtx         sync.Mutex
	activeSpans uint32
	closed      bool
}

type Span struct {
	SpanID       uint64            `json:"i,omitempty"`
	ParentSpanID uint64            `json:"p,omitempty"`
	Failed       bool              `json:"f,omitempty"`
	Begin        int64             `json:"b,omitempty"`
	End          int64             `json:"e,omitempty"`
	Metadata     map[string]string `json:"m,omitempty"`
	LogEntries   []LogEntry        `json:"l,omitempty"`

	mtx    sync.Mutex
	trace  *Trace
	config config
}

type LogEntry struct {
	Time  int64  `json:"t,omitempty"`
	Text  string `json:"m,omitempty"`
	Stack string `json:"s,omitempty"`
	Error bool   `json:"e,omitempty"`
}

type spanKey struct{}
type traceKey struct{}
type tracerKey struct{}

func Get(ctx context.Context) *Span {
	span, ok := ctx.Value(spanKey{}).(*Span)
	if ok && span != nil {
		return span
	}

	return nil
}

type config struct {
	NewTrace   bool
	PanicGuard bool
}

type TraceOption func(*config)

var WithNewTrace = TraceOption(withNewTrace)
var WithPanicGuard = TraceOption(withPanicGuard)

func withNewTrace(c *config) {
	c.NewTrace = true
}

func withPanicGuard(c *config) {
	c.PanicGuard = true
}

func New(ctx context.Context, name string, options ...TraceOption) (*Span, context.Context) {
	var config config
	for _, opt := range options {
		opt(&config)
	}

	if !config.NewTrace {
		span, ok := ctx.Value(spanKey{}).(*Span)
		if ok && span != nil {
			sub := span.subSpan(name, config)
			ctx = context.WithValue(ctx, spanKey{}, sub)
			return sub, ctx
		}

		trace, ok := ctx.Value(traceKey{}).(*Trace)
		if ok && trace != nil {
			sub := trace.root.subSpan(name, config)
			ctx = context.WithValue(ctx, spanKey{}, sub)
			return sub, ctx
		}
	}

	tracer, ok := ctx.Value(tracerKey{}).(*Tracer)
	if !ok || tracer == nil {
		tracer = DefaultTracer()
	}

	span := tracer.trace(name, config)
	ctx = context.WithValue(ctx, traceKey{}, span.trace)
	ctx = context.WithValue(ctx, spanKey{}, span)
	return span, ctx
}

func (t *Trace) spanDone() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.closed {
		panic("too many calls to spanDone()")
	}

	t.activeSpans--

	if t.activeSpans == 0 {
		t.closed = true
		t.tracer.buffer <- t
	}
}

func (t *Trace) spanAdd(span *Span) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.closed {
		panic("trace is already closed")
	}

	t.activeSpans++
	t.Spans = append(t.Spans, span)
}

func (t *Trace) rootSpan(name string, c config) *Span {
	root := getSpan()
	root.SpanID = atomic.AddUint64(&t.nextSpanID, 1)
	root.Begin = time.Now().UnixNano()
	root.trace = t
	root.config = c
	if name != "" {
		root.Metadata["trace/name"] = name
	}

	t.root = root
	t.spanAdd(root)
	return root
}

func (s *Span) subSpan(name string, c config) *Span {
	sub := getSpan()
	sub.SpanID = atomic.AddUint64(&s.trace.nextSpanID, 1)
	sub.ParentSpanID = s.SpanID
	sub.Begin = time.Now().UnixNano()
	sub.trace = s.trace
	sub.config = c
	if name != "" {
		sub.Metadata["trace/name"] = name
	}

	s.trace.spanAdd(sub)
	return sub
}

func (s *Span) addLogEntry(e LogEntry) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.End != 0 {
		panic("trace span is already closed")
	}

	if e.Error {
		s.Failed = true
	}

	s.LogEntries = append(s.LogEntries, e)
}

func (s *Span) TraceID() string {
	return s.trace.ID
}

func (s *Span) Log(args ...interface{}) {
	s.addLogEntry(LogEntry{
		Time: time.Now().UnixNano(),
		Text: fmt.Sprint(args...),
	})
}

func (s *Span) Logf(format string, args ...interface{}) {
	s.addLogEntry(LogEntry{
		Time: time.Now().UnixNano(),
		Text: fmt.Sprintf(format, args...),
	})
}

func (s *Span) Error(args ...interface{}) error {
	var (
		err  error
		text string
	)

	if len(args) == 1 {
		if e, ok := args[0].(error); ok && e != nil {
			err = e
			text = e.Error()
		}
	}

	if err == nil {
		text = fmt.Sprint(args...)
		err = errors.New(text)
	}

	s.addLogEntry(LogEntry{
		Time:  time.Now().UnixNano(),
		Text:  text,
		Stack: string(debug.Stack()),
		Error: true,
	})

	return err
}

func (s *Span) Errorf(format string, args ...interface{}) error {
	var (
		text = fmt.Sprintf(format, args...)
		err  = errors.New(text)
	)

	s.addLogEntry(LogEntry{
		Time:  time.Now().UnixNano(),
		Text:  text,
		Stack: string(debug.Stack()),
		Error: true,
	})

	return err
}

func (s *Span) Fatal(args ...interface{}) {
	e := LogEntry{
		Time:  time.Now().UnixNano(),
		Text:  fmt.Sprint(args...),
		Stack: string(debug.Stack()),
		Error: true,
	}
	s.addLogEntry(e)
	panic(&e)
}

func (s *Span) Fatalf(format string, args ...interface{}) {
	e := LogEntry{
		Time:  time.Now().UnixNano(),
		Text:  fmt.Sprintf(format, args...),
		Stack: string(debug.Stack()),
		Error: true,
	}
	s.addLogEntry(e)
	panic(&e)
}

func (s *Span) Close() {
	var (
		repanic      bool
		repanicValue interface{}
	)

	if r := recover(); r != nil {
		e, ok := r.(*LogEntry)
		if ok {
			r = e.Text
		} else {
			s.addLogEntry(LogEntry{
				Time:  time.Now().UnixNano(),
				Text:  fmt.Sprintf("panic: %v", r),
				Stack: string(debug.Stack()),
				Error: true,
			})
		}
		if !s.config.PanicGuard {
			// re-panic
			repanic = true
			repanicValue = r
		}
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.End == 0 {
		s.End = time.Now().UnixNano()
		s.trace.spanDone()
	}

	if repanic {
		panic(repanicValue)
	}
}
