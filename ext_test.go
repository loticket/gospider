package gospider

import (
	"github.com/stretchr/testify/assert"
	"github.com/zhshch2002/goreq"
	"testing"
)

func TestWithRobotsTxt(t *testing.T) {
	s := NewSpider(WithRobotsTxt("gospider"))
	s.SeedTask(goreq.Get("https://github.com/gist/"), func(ctx *Context) {
		t.Error("RobotsTxt error")
	})
	got := false
	s.SeedTask( // unable to access according to https://github.com/robots.txt
		goreq.Get("https://github.com/zhshch2002/goribot/wiki"),
		func(ctx *Context) {
			got = true
		},
	)
	s.Wait()
	assert.True(t, got)
}

func TestWithDepthLimit(t *testing.T) {
	s := NewSpider(WithDepthLimit(2))
	s.SeedTask(goreq.Get("https://httpbin.org/get"), func(ctx *Context) {
		ctx.Println("Depth", ctx.Req.Context().Value("depth")) // 1
		ctx.AddTask(goreq.Get("https://httpbin.org/get"), func(ctx *Context) {
			ctx.Println("Depth", ctx.Req.Context().Value("depth")) // 2
			ctx.AddTask(goreq.Get("https://httpbin.org/get"), func(ctx *Context) {
				ctx.Println("Depth", ctx.Req.Context().Value("depth")) // 3
				t.Error("Limiter error")
			})
		})
	})
	s.Wait()
}

func TestWithMaxReqLimit(t *testing.T) {
	s := NewSpider(WithMaxReqLimit(2))
	count := 0
	s.SeedTask(goreq.Get("https://httpbin.org/get"), func(ctx *Context) {
		count += 1
	})
	s.SeedTask(goreq.Get("https://httpbin.org/get"), func(ctx *Context) {
		count += 1
	})
	s.SeedTask(goreq.Get("https://httpbin.org/get"), func(ctx *Context) {
		count += 1
	})
	s.Wait()
	assert.Equal(t, 2, count)
}
