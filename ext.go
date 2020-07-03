package gospider

import (
	"context"
	"github.com/slyrz/robots"
	"github.com/zhshch2002/goreq"
	"strings"
	"sync/atomic"
)

func WithRobotsTxt(ua string) Extension {
	return func(s *Spider) {
		rs := map[string]*robots.Robots{}
		s.OnTask(func(ctx *Context, t *Task) *Task {
			var r *robots.Robots
			if a, ok := rs[t.Req.URL.Host]; ok {
				r = a
			} else {
				if u, err := t.Req.URL.Parse("/robots.txt"); err == nil {
					if resp, err := goreq.Get(u.String()).Do().Resp(); err == nil && resp.StatusCode == 200 {
						r = robots.New(strings.NewReader(resp.Text), ua)
						rs[t.Req.URL.Host] = r
					}
				}
			}
			if r != nil {
				if !r.Allow(t.Req.URL.Path) {
					return nil
				}
			}
			return t
		})
	}
}

func WithDepthLimit(max int) Extension {
	return func(s *Spider) {
		s.OnTask(func(ctx *Context, t *Task) *Task {
			if ctx.Req == nil || ctx.Req.Context().Value("depth") == nil {
				t.Req.Request = t.Req.WithContext(context.WithValue(t.Req.Context(), "depth", 1))
				return t
			} else {
				depth := ctx.Req.Context().Value("depth").(int)
				if depth < max {
					t.Req.Request = t.Req.WithContext(context.WithValue(t.Req.Context(), "depth", depth+1))
					return t
				} else {
					return nil
				}
			}
		})
	}
}

func WithMaxReqLimit(max int64) Extension {
	return func(s *Spider) {
		count := int64(0)
		s.OnTask(func(ctx *Context, t *Task) *Task {
			if count < max {
				atomic.AddInt64(&count, 1)
				return t
			}
			return nil
		})
	}
}
