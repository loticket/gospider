package gospider

import (
	"context"
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/slyrz/robots"
	"github.com/loticket/goreq"
	"io"
	"strings"
	"sync"
	"sync/atomic"
)

func WithDeduplicate() Extension {
	return func(s *Spider) {
		CrawledHash := map[[md5.Size]byte]struct{}{}
		lock := sync.Mutex{}
		s.OnTask(func(ctx *Context, t *Task) *Task {
			has := GetRequestHash(t.Req)
			lock.Lock()
			defer lock.Unlock()
			if _, ok := CrawledHash[has]; ok {
				return nil
			}
			CrawledHash[has] = struct{}{}
			return t
		})
	}

}

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

func WithErrorLog(f io.Writer) Extension {
	return func(s *Spider) {
		l := zerolog.New(f).With().Timestamp().Logger()
		send := func(ctx *Context, err error, t, stack string) {
			event := l.Err(err).
				Str("spider", s.Name).
				Str("type", "item").
				Str("ctx", fmt.Sprint(ctx)).
				Str("url", ctx.Req.URL.String()).
				AnErr("req err", ctx.Req.Err).
				AnErr("resp err", ctx.Resp.Err)
			if ctx.Resp != nil {
				event.Int("resp code", ctx.Resp.StatusCode)
				if ctx.Resp.Text != "" {
					event.Str("text", ctx.Resp.Text)
				}
			}
			event.Str("stack", SprintStack()).Send()
		}

		s.OnItem(func(ctx *Context, i interface{}) interface{} {
			if err, ok := i.(error); ok {
				send(ctx, err, "item", SprintStack())
			}
			return i
		})
		s.OnRecover(func(ctx *Context, err error) {
			send(ctx, err, "OnRecover", SprintStack())
		})
		s.OnReqError(func(ctx *Context, err error) {
			send(ctx, err, "OnReqError", SprintStack())
		})
		s.OnRespError(func(ctx *Context, err error) {
			send(ctx, err, "OnRespError", SprintStack())
		})
	}
}

type CsvItem []string

func WithCsvItemSaver(f io.Writer) Extension {
	lock := sync.Mutex{}
	w := csv.NewWriter(f)
	return func(s *Spider) {
		s.OnItem(func(ctx *Context, i interface{}) interface{} {
			if data, ok := i.(CsvItem); ok {
				lock.Lock()
				defer lock.Unlock()
				err := w.Write(data)
				if err != nil {
					log.Err(err).Msg("WithCsvItemSaver Error")
				}
				w.Flush()
			}
			return i
		})
	}
}
