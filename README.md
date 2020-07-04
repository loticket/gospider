# Gospider
[![codecov](https://codecov.io/gh/zhshch2002/gospider/branch/master/graph/badge.svg)](https://codecov.io/gh/zhshch2002/gospider)

轻量的 Golang 爬虫框架。[Github](https://github.com/zhshch2002/gospider)

## 🚀Feature
* 优雅的 API
* 整洁的文档
* 高速（单核处理 >1K task/sec）
* 友善的分布式支持
* 一些细节
  * 相对链接自动转换
  * 字符编码自动解码
  * HTML,JSON 自动解析
* 丰富的扩展支持
  * 自动去重
  * 失败重试
  * 记录异常请求
  * 控制延时、随机延时、并发、速率
  * Robots.txt 支持
  * 随机 UA
* 轻量，适于学习或快速开箱搭建

## 👜获取 Goribot
```sh
go get -u github.com/zhshch2002/gospider
```

Gospider 从 [Goribot](https://github.com/zhshch2002/goribot) 改进而来，解决了队列任务丢失等问题，可以参考原项目的一些文档。

Gospider 将网络请求部分移动到 [Goreq|https://github.com/zhshch2002/goreq](https://github.com/zhshch2002/goribot) 单独管理。

## ⚡建立你的第一个项目
```go
package main

import (
	"github.com/zhshch2002/goreq"
	"github.com/zhshch2002/gospider"
)

func main() {
    s := gospider.NewSpider() // 创建蜘蛛

    s.SeedTask( // 种子任务
        goreq.Get("https://httpbin.org/get"),
        func(ctx *gospider.Context) {
            ctx.AddItem(ctx.Resp.Text) // 提交任务爬取结果
        },
    )
    s.OnItem(func(ctx *gospider.Context, i interface{}) interface{} { // 收集并存储结果
        ctx.Println(i)
        return i
    })

    s.Wait() // 等待所有任务完成并释放资源
}
```

## 从 Colly 了解 Gospider
```go
package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/zhshch2002/goreq"
	"github.com/zhshch2002/gospider"
)

/* colly example http://go-colly.org/docs/examples/basic/

// Instantiate default collector
c := colly.NewCollector(
	// Visit only domains: hackerspaces.org, wiki.hackerspaces.org
	colly.AllowedDomains("hackerspaces.org", "wiki.hackerspaces.org"),
)

// On every a element which has href attribute call callback
c.OnHTML("a[href]", func(e *colly.HTMLElement) {
	link := e.Attr("href")
	// Print link
	fmt.Printf("Link found: %q -> %s\n", e.Text, link)
	// Visit link found on page
	// Only those links are visited which are in AllowedDomains
	c.Visit(e.Request.AbsoluteURL(link))
})

// Before making a request print "Visiting ..."
c.OnRequest(func(r *colly.Request) {
	fmt.Println("Visiting", r.URL.String())
})

// Start scraping on https://hackerspaces.org
c.Visit("https://hackerspaces.org/")
*/
func main() {
	s := gospider.NewSpider(goreq.WithFilterLimiter(false, &goreq.FilterLimiterOpinion{
		LimiterMatcher: goreq.LimiterMatcher{Glob: "*.hackerspaces.org"},
		Allow:          true,
	}, &goreq.FilterLimiterOpinion{
		LimiterMatcher: goreq.LimiterMatcher{Glob: "hackerspaces.org"},
		Allow:          true,
	}))

	// On every a element which has href attribute call callback
	s.OnHTML("a[href]", func(ctx *gospider.Context, sel *goquery.Selection) {
		link, _ := sel.Attr("href")
		// Print link
		ctx.Printf("Link found: %q -> %s\n", sel.Text(), link)
		// Visit link found on page
		// Only those links are visited which are in AllowedDomains
		ctx.AddTask(goreq.Get(link)) // gospider will auto convert to absolute URL
	})

	s.SeedTask(goreq.Get("https://hackerspaces.org/"))
	s.Wait()
}
```

## 详细介绍
### Spider
```go
// 创建新的蜘蛛
s := gospider.NewSpider()

// 出入的参数是 type Extension func(s *Spider) 或者 goreq 的 type Middleware func(*Client, Handler) Handler
// 本质上是两个函数用于对蜘蛛进行扩展
func NewSpider(e ...interface{}) *Spider
func (s *Spider) Use(exts ...interface{})
```

蜘蛛自创建起就已经开始运行。蜘蛛提供了一系列回调函数以在不同生命周期进行处理。

```go
package main

import (
	"github.com/zhshch2002/goreq"
	"github.com/zhshch2002/gospider"
)

func main() {
	s := gospider.NewSpider()

	// 当新的任务被执行前（返回 nil 以取消任务）
	s.OnTask(func(ctx *gospider.Context, t *gospider.Task) *gospider.Task {
		ctx.Println("OnTask")
		return t
	})
	// 收到响应时
	s.OnResp(func(ctx *gospider.Context) {
		ctx.Println("OnResp")
	})
	// 处理通过`ctx.AddItem()`提交的结果（返回 nil 以中断多个回调函数连续处理），独立处理以减小对网络处理的阻塞
	s.OnItem(func(ctx *gospider.Context, i interface{}) interface{} {
		ctx.Println("OnItem", i)
		return i
	})
	// 在蜘蛛执行中出现 panic
	s.OnRecover(func(ctx *gospider.Context, err error) {
		ctx.Println("OnRecover", err)
	})
	// 在创建新的 requests 时出现错误
	s.OnReqError(func(ctx *gospider.Context, err error) {
		ctx.Println("OnReqError", err)
	})
	// 网络请求出现错误时
	s.OnRespError(func(ctx *gospider.Context, err error) {
		ctx.Println("OnRespError", err)
	})

	// 创建种子任务
	s.SeedTask(
		goreq.Get("https://httpbin.org/get"),
		func(ctx *gospider.Context) { // 与此任务绑定的回调函数，等同于针对这个请求的 OnResp。
			ctx.AddTask(goreq.Get("https://httpbin.org/get")) // 使用 ctx 创建的任务可以记录上一个请求的信息，再由其他扩展添加 Referer 等信息。
			ctx.AddItem(ctx.Resp.Text)
		},
	)
	s.SeedTask(goreq.Get("htps://httpbin.org/get"))
	s.Wait() // 等待所有任务执行完成并释放资源
}
```

### Request 和 Response
网络部分请看 [Goreq](https://github.com/zhshch2002/goreq) 的 [文档](https://wiki.imagician.net/goreq/)。

请求可以被链式配置，如果配置过程中出现错误，`req.Err`将不再为`nil`
* `AddParam(k, v string)`
* `AddParams(v map[string]string)`
* `AddHeader(key, value string)`
* `AddHeaders(v map[string]string)`
* `AddCookie(c *http.Cookie)`
* `SetUA(ua string)`
* `SetBasicAuth(username, password string)`
* `SetProxy(urladdr string)`
* Set request body data
    * `SetBody(b io.Reader)` basic setting
    * `SetRawBody(b []byte)`
    * `SetFormBody(v map[string]string)`
    * `SetJsonBody(v interface{})`
    * `SetMultipartBody(data ...interface{})` Set a slice of `FormField` and `FormFile` struct as body data
* `Callback(fn func(resp *Response)` Set a callback func run after req `Do()`

`*goreq.Response`可以通过下述函数来获取响应数据。
* `Resp() (*Response, error)` 获取响应本身以及网络请求错误。
* `Txt() (string, error)` 自动处理完编码并解析为文本后的内容以及网络请求错误。
* `HTML() (*goquery.Document, error)`
* `XML() (*xmlpath.Node, error)`
* `BindXML(i interface{}) error`
* `JSON() (gjson.Result, error)`
* `BindJSON(i interface{}) error`
* `Error() error` 网络请求错误。（正常情况下为`nil`）