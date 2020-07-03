package gospider

import (
	"bytes"
	"context"
	"crypto/md5"
	"github.com/slyrz/robots"
	"github.com/zhshch2002/goreq"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
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
	tmpl, err := template.New("ErrorLog").Parse(`--------------
Error:     {{.err}}
Spider:    {{.s.Name}}
Type:      {{.type}}
URL:       {{.ctx.Req.URL}}
ReqError:  {{.ctx.Req.Err}}
RespError: {{.ctx.Resp.Err}}
{{if .ctx.Resp}}RespCode:  {{.ctx.Resp.StatusCode}}
{{if .ctx.Resp.Text}}Text:
{{.ctx.Resp.Text}}{{end}}{{end}}

Stack:     {{.stack}}
--------------

`)
	if err != nil {
		panic(err)
	}
	return func(s *Spider) {
		formatError := func(ctx *Context, err error, t, stack string) string {
			buf := bytes.NewBuffer([]byte{})
			err = tmpl.Execute(buf, map[string]interface{}{
				"ctx":   ctx,
				"s":     s,
				"err":   err,
				"type":  t,
				"stack": stack,
			})
			if err != nil {
				log.Println("[WithErrorLog]", err)
			}
			return buf.String()
		}
		lock := sync.Mutex{}
		s.OnItem(func(ctx *Context, i interface{}) interface{} {
			if err, ok := i.(error); ok {
				lock.Lock()
				defer lock.Unlock()
				_, err := f.Write([]byte(formatError(ctx, err, "OnItem", SprintStack())))
				if err != nil {
					log.Println("[WithErrorLog]", err)
				}
			}
			return i
		})
		s.OnRecover(func(ctx *Context, err error) {
			lock.Lock()
			defer lock.Unlock()
			_, e := f.Write([]byte(formatError(ctx, err, "OnRecover", SprintStack())))
			if e != nil {
				log.Println("[WithErrorLog]", e)
			}
		})
		s.OnReqError(func(ctx *Context, err error) {
			lock.Lock()
			defer lock.Unlock()
			_, e := f.Write([]byte(formatError(ctx, err, "OnReqError", SprintStack())))
			if e != nil {
				log.Println("[WithErrorLog]", e)
			}
		})
		s.OnRespError(func(ctx *Context, err error) {
			lock.Lock()
			defer lock.Unlock()
			_, e := f.Write([]byte(formatError(ctx, err, "OnRespError", SprintStack())))
			if e != nil {
				log.Println("[WithErrorLog]", e)
			}
		})
	}
}
