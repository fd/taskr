package router

import (
	"fmt"
	"regexp"
	"unsafe"
)

type instMatchByte struct {
	b          byte
	frameCount int
	onErr      jumpPointer
}

func (i *instMatchByte) Exec(ctx *runtime) {
	if i.b == ctx.cur {
		ctx.Next()
	} else {
		ctx.Jump(i.onErr)
	}
}

func (i *instMatchByte) MemorySize() int {
	if i == nil {
		return 0
	}
	const size = int(unsafe.Sizeof(instMatchByte{}))
	return size
}

type instMatchBytes struct {
	b          []byte
	frameCount int
	onErr      jumpPointer
}

func (i *instMatchBytes) Exec(ctx *runtime) {
	for _, b := range i.b {
		if ctx.end {
			ctx.Jump(i.onErr)
			return
		}
		if b == ctx.cur {
			ctx.Next()
		} else {
			ctx.Jump(i.onErr)
			return
		}
	}
}

func (i *instMatchBytes) MemorySize() int {
	if i == nil {
		return 0
	}
	const size = int(unsafe.Sizeof(instMatchBytes{}))
	return size + len(i.b)
}

type instMatchVariable struct {
	names      []string
	pattern    *regexp.Regexp
	min, max   int
	frameCount int
	onErr      jumpPointer
}

func (i *instMatchVariable) MemorySize() int {
	if i == nil {
		return 0
	}
	const size = int(unsafe.Sizeof(instMatchVariable{}))
	s := size
	s += len(i.names) * 16
	for _, n := range i.names {
		s += len(n)
	}

	return s
}

func (i *instMatchVariable) Exec(ctx *runtime) {
	var (
		n   int
		beg = ctx.offset
	)

LOOP:
	for {
		if ctx.end {
			break LOOP
		}

		if ctx.cur == '/' {
			val := ctx.path[beg:ctx.offset]
			if !i.matchPattern(ctx, val) {
				break LOOP
			}
			i.addParam(ctx, val)

			n++
			if n >= i.max && i.max > 0 {
				beg = ctx.offset
				break LOOP
			}

		DELIM:
			for {
				if ctx.end {
					break DELIM
				}

				switch ctx.cur {

				case '/':
					ctx.Next()
					beg = ctx.offset

				case '.':
					ctx.Next()

				default:
					break DELIM

				}
			}
			ctx.SetOffset(beg)

		} else {
			ctx.Next()

		}

	}

	if ctx.offset != beg {
		val := ctx.path[beg:ctx.offset]
		if i.matchPattern(ctx, val) {
			i.addParam(ctx, val)
			n++
		}
	}

	if n < i.min {
		ctx.Jump(i.onErr)
		return
	}
	if n > i.max && i.max > 0 {
		ctx.Jump(i.onErr)
		return
	}
}

func (i *instMatchVariable) matchPattern(ctx *runtime, val string) bool {
	if i.pattern == nil {
		return true
	}

	if i.pattern.MatchString(val) {
		return true
	}

	return false
}

func (i *instMatchVariable) addParam(ctx *runtime, val string) {
	for _, name := range i.names {
		ctx.AddParam(name, val)
	}
}

type instMatchEpsilon struct {
	frameCount int
	onErr      jumpPointer
}

func (i *instMatchEpsilon) MemorySize() int {
	if i == nil {
		return 0
	}
	const size = int(unsafe.Sizeof(instMatchEpsilon{}))
	return size
}

func (i *instMatchEpsilon) Exec(ctx *runtime) {
	idx := -1

LOOP:
	for {
		if ctx.end {
			break LOOP
		}
		switch ctx.cur {

		case '/':
			ctx.Next()
			idx = ctx.offset

		case '.':
			if idx == -1 {
				break LOOP
			}
			ctx.Next()

		default:
			break LOOP

		}
	}

	if !ctx.end {
		if idx == -1 {
			ctx.Jump(i.onErr)
			return
		}

		ctx.SetOffset(idx)
	}
}

type instMatchEnd struct {
	handlers   []handler
	frameCount int
	jump       jumpPointer
}

func (i *instMatchEnd) MemorySize() int {
	if i == nil {
		return 0
	}
	const size = int(unsafe.Sizeof(instMatchEnd{}))
	return size + (len(i.handlers) * 16)
}

func (i *instMatchEnd) Exec(ctx *runtime) {

LOOP:
	for {
		if ctx.end {
			break LOOP
		}
		switch ctx.cur {

		case '/':
			ctx.Next()

		default:
			break LOOP

		}
	}

	if ctx.end {
		ctx.Commit(i.handlers)
	}

	ctx.Jump(i.jump)
}

func (i *instMatchByte) String() string {
	return fmt.Sprintf("matchByte(%q, frames: %d, onErr: %d)", i.b, i.frameCount, i.onErr)
}

func (i *instMatchBytes) String() string {
	return fmt.Sprintf("matchBytes(%q, frames: %d, onErr: %d)", i.b, i.frameCount, i.onErr)
}

func (i *instMatchVariable) String() string {
	return fmt.Sprintf("matchVariable(frames: %d, onErr: %d)", i.frameCount, i.onErr)
}

func (i *instMatchEpsilon) String() string {
	return fmt.Sprintf("matchEpsilon(frames: %d, onErr: %d)", i.frameCount, i.onErr)
}

func (i *instMatchEnd) String() string {
	return fmt.Sprintf("matchEnd(frames: %d, jump: %d)", i.frameCount, i.jump)
}

func (i *instMatchByte) Frames() int {
	return i.frameCount
}

func (i *instMatchBytes) Frames() int {
	return i.frameCount
}

func (i *instMatchVariable) Frames() int {
	return i.frameCount
}

func (i *instMatchEpsilon) Frames() int {
	return i.frameCount
}

func (i *instMatchEnd) Frames() int {
	return i.frameCount
}

func (i *instMatchByte) Jump() *jumpPointer {
	return &i.onErr
}

func (i *instMatchBytes) Jump() *jumpPointer {
	return &i.onErr
}

func (i *instMatchVariable) Jump() *jumpPointer {
	return &i.onErr
}

func (i *instMatchEpsilon) Jump() *jumpPointer {
	return &i.onErr
}

func (i *instMatchEnd) Jump() *jumpPointer {
	return &i.jump
}
