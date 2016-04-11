package trace

import (
	"log"
	"sync"

	"github.com/pborman/uuid"
)

var DefaultHandler Handler
var fallbackHandler Handler = &fallbackHandlerImp{}
var defaultTracer *Tracer
var defaultTracerInitOnce sync.Once

func DefaultTracer() *Tracer {
	defaultTracerInitOnce.Do(defaultTracerInit)
	return defaultTracer
}

func defaultTracerInit() {
	defaultTracer = NewTracer(DefaultHandler)
}

type Handler interface {
	HandleTrace(*Trace) error
}

type Tracer struct {
	handler Handler
	buffer  chan *Trace
}

func NewTracer(handler Handler) *Tracer {
	if handler == nil {
		handler = DefaultHandler
	}
	if handler == nil {
		handler = fallbackHandler
	}

	t := &Tracer{
		handler: handler,
		buffer:  make(chan *Trace, 512),
	}

	go t.runTraceSender()

	return t
}

func (t *Tracer) trace(name string, c config) *Span {
	trace := getTrace()
	trace.ID = uuid.New()
	trace.tracer = t

	span := trace.rootSpan(name, c)
	return span
}

func (t *Tracer) Trace(name string, options ...TraceOption) *Span {
	var config config
	for _, opt := range options {
		opt(&config)
	}

	return t.trace(name, config)
}

func (t *Tracer) runTraceSender() {
	for trace := range t.buffer {
		t.handleTrace(trace)
	}
}

func (t *Tracer) handleTrace(trace *Trace) {
	err := t.handler.HandleTrace(trace)
	if err != nil {
		log.Printf("trace: error while handling trace: %s", err)
	}
}

type fallbackHandlerImp struct{}

func (h *fallbackHandlerImp) HandleTrace(t *Trace) error { return nil }
