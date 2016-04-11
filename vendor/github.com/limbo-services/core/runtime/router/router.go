package router

import (
	"net/http"
	"sync"
	"unsafe"

	"golang.org/x/net/context"
)

// Router dispatches a request to the correct handlers
type Router struct {
	mtx     sync.RWMutex
	routes  []route
	program []instruction
	err     error
}

// Params holds the parameters for the current route
type Params struct {
	params []param
}

type route struct {
	pattern string
	handler Handler
}

type paramsKeyType string

const paramsKey = paramsKeyType("params")

// P returns the Params for the current context
func P(ctx context.Context) Params {
	x, _ := ctx.Value(paramsKey).([]param)
	return Params{x}
}

// Get the first value for the provided key
func (p Params) Get(key string) string {
	for _, p := range p.params {
		if p.name == key {
			return p.value
		}
	}

	return ""
}

// GetAll values for the provided key
func (p Params) GetAll(key string) []string {
	var s = make([]string, 0, 8)

	for _, p := range p.params {
		if p.name == key {
			s = append(s, p.value)
		}
	}

	return s
}

// Add a route to the router
func (r *Router) Add(method, pattern string, handlers ...Handler) *Router {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.program = nil
	r.err = nil

	for _, h := range handlers {
		r.routes = append(r.routes, route{pattern, whenMethodIs(method, h)})
	}

	return r
}

// Addf a route to the router
func (r *Router) Addf(method, pattern string, handlers ...HandlerFunc) *Router {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.program = nil
	r.err = nil

	for _, h := range handlers {
		r.routes = append(r.routes, route{pattern, whenMethodIs(method, h)})
	}

	return r
}

func (r *Router) ANY(pattern string, handlers ...Handler) *Router {
	return r.Add("*", pattern, handlers...)
}

func (r *Router) ANYf(pattern string, handlers ...HandlerFunc) *Router {
	return r.Addf("*", pattern, handlers...)
}

func (r *Router) GET(pattern string, handlers ...Handler) *Router {
	return r.Add("GET", pattern, handlers...)
}

func (r *Router) GETf(pattern string, handlers ...HandlerFunc) *Router {
	return r.Addf("GET", pattern, handlers...)
}

func (r *Router) POS(pattern string, handlers ...Handler) *Router {
	return r.Add("POST", pattern, handlers...)
}

func (r *Router) POSf(pattern string, handlers ...HandlerFunc) *Router {
	return r.Addf("POST", pattern, handlers...)
}

func (r *Router) PAT(pattern string, handlers ...Handler) *Router {
	return r.Add("PATCH", pattern, handlers...)
}

func (r *Router) PATf(pattern string, handlers ...HandlerFunc) *Router {
	return r.Addf("PATCH", pattern, handlers...)
}

func (r *Router) PUT(pattern string, handlers ...Handler) *Router {
	return r.Add("PUT", pattern, handlers...)
}

func (r *Router) PUTf(pattern string, handlers ...HandlerFunc) *Router {
	return r.Addf("PUT", pattern, handlers...)
}

func (r *Router) DEL(pattern string, handlers ...Handler) *Router {
	return r.Add("DELETE", pattern, handlers...)
}

func (r *Router) DELf(pattern string, handlers ...HandlerFunc) *Router {
	return r.Addf("DELETE", pattern, handlers...)
}

type Filter interface {
	ApplyFilter(Handler) Handler
}

type FilterFunc func(Handler) Handler

func (f FilterFunc) ApplyFilter(handler Handler) Handler {
	return f(handler)
}

type FilteredRouter struct {
	router  *Router
	filters []Filter
}

func (r *Router) Filter(filters ...Filter) *FilteredRouter {
	return &FilteredRouter{
		router:  r,
		filters: filters,
	}
}

func (r *Router) Filterf(filters ...FilterFunc) *FilteredRouter {
	var f = make([]Filter, len(filters))
	for i, ff := range filters {
		f[i] = ff
	}

	return &FilteredRouter{
		router:  r,
		filters: f,
	}
}

func (r *FilteredRouter) Filter(filters ...Filter) *FilteredRouter {
	return &FilteredRouter{
		router:  r.router,
		filters: append(r.filters, filters...),
	}
}

func (r *FilteredRouter) Filterf(filters ...FilterFunc) *FilteredRouter {
	var f = make([]Filter, len(filters))
	for i, ff := range filters {
		f[i] = ff
	}

	return &FilteredRouter{
		router:  r.router,
		filters: append(r.filters, f...),
	}
}

// Add a route to the router
func (r *FilteredRouter) Add(method, pattern string, handlers ...Handler) *FilteredRouter {
	for i, handler := range handlers {
		handlers[i] = r.applyFilters(handler)
	}
	r.router.Add(method, pattern, handlers...)
	return r
}

// Addf a route to the router
func (r *FilteredRouter) Addf(method, pattern string, handlers ...HandlerFunc) *FilteredRouter {
	var h = make([]Handler, len(handlers))
	for i, handler := range handlers {
		h[i] = r.applyFilters(handler)
	}
	r.router.Add(method, pattern, h...)
	return r
}

func (r *FilteredRouter) ANY(pattern string, handlers ...Handler) *FilteredRouter {
	return r.Add("*", pattern, handlers...)
}

func (r *FilteredRouter) ANYf(pattern string, handlers ...HandlerFunc) *FilteredRouter {
	return r.Addf("*", pattern, handlers...)
}

func (r *FilteredRouter) GET(pattern string, handlers ...Handler) *FilteredRouter {
	return r.Add("GET", pattern, handlers...)
}

func (r *FilteredRouter) GETf(pattern string, handlers ...HandlerFunc) *FilteredRouter {
	return r.Addf("GET", pattern, handlers...)
}

func (r *FilteredRouter) POS(pattern string, handlers ...Handler) *FilteredRouter {
	return r.Add("POST", pattern, handlers...)
}

func (r *FilteredRouter) POSf(pattern string, handlers ...HandlerFunc) *FilteredRouter {
	return r.Addf("POST", pattern, handlers...)
}

func (r *FilteredRouter) PAT(pattern string, handlers ...Handler) *FilteredRouter {
	return r.Add("PATCH", pattern, handlers...)
}

func (r *FilteredRouter) PATf(pattern string, handlers ...HandlerFunc) *FilteredRouter {
	return r.Addf("PATCH", pattern, handlers...)
}

func (r *FilteredRouter) PUT(pattern string, handlers ...Handler) *FilteredRouter {
	return r.Add("PUT", pattern, handlers...)
}

func (r *FilteredRouter) PUTf(pattern string, handlers ...HandlerFunc) *FilteredRouter {
	return r.Addf("PUT", pattern, handlers...)
}

func (r *FilteredRouter) DEL(pattern string, handlers ...Handler) *FilteredRouter {
	return r.Add("DELETE", pattern, handlers...)
}

func (r *FilteredRouter) DELf(pattern string, handlers ...HandlerFunc) *FilteredRouter {
	return r.Addf("DELETE", pattern, handlers...)
}

func (r *FilteredRouter) applyFilters(handler Handler) Handler {
	for i := len(r.filters) - 1; i >= 0; i-- {
		filter := r.filters[i]
		handler = filter.ApplyFilter(handler)
	}
	return handler
}

// Compile the router rules
func (r *Router) Compile() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.program != nil {
		return nil
	}

	c := compiler{}
	for _, route := range r.routes {
		err := c.Insert(route.pattern, route.handler)
		if err != nil {
			r.err = err
			return err
		}
	}

	c.Optimize()
	r.program = c.Compile()
	return nil
}

func (r *Router) getProgram() ([]instruction, error) {
	r.mtx.RLock()
	program, err := r.program, r.err
	r.mtx.RUnlock()

	if err != nil {
		return nil, err
	}

	if program != nil {
		return program, nil
	}

	r.Compile()
	return r.getProgram()
}

func (r *Router) ServeHTTP(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
	program, err := r.getProgram()
	if err != nil {
		panic(err)
	}

	return runtimeExec(program, ctx, rw, req)
}

func (r *Router) MemorySize() int {
	if r == nil {
		return 0
	}
	const sizeRouter = int(unsafe.Sizeof(Router{}))
	const sizeRoute = int(unsafe.Sizeof(route{}))

	s := sizeRouter
	s += len(r.routes) * sizeRoute
	s += len(r.program) * 16
	for _, r := range r.routes {
		s += len(r.pattern)
	}
	for _, i := range r.program {
		s += i.MemorySize()
	}

	return s
}
