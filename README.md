# Gospider
[![codecov](https://codecov.io/gh/zhshch2002/gospider/branch/master/graph/badge.svg)](https://codecov.io/gh/zhshch2002/gospider)

[中文文档](https://gospider.athorx.com/)

`Gospider`是一个轻量的，对分布式有好的Go爬虫框架。`Goreq`是同时设计的一个基于 Go 标准库`net/http`的包装库，用来提供简单的`Http`访问操作。

`Gospider`得益于Go便捷的协程，具有极高的效率。同时提供类似`colly`和`scrapy`两种处理方式。

* Goreq - https://github.com/zhshch2002/goreq

## 🚀Feature

* **优雅的 API**
* **整洁的文档**
* **高速（单核处理 >1K task/sec）**
* **友善的分布式支持**
* **一些细节** 相对链接自动转换、字符编码自动解码、HTML,JSON 自动解析
* **丰富的扩展支持** 自动去重、失败重试、记录异常请求、控制延时、随机延时、并发、速率、Robots.txt 支持、随机 UA
* **轻量，适于学习或快速开箱搭建**



## ⚡网络请求

```shell
go get -u github.com/zhshch2002/goreq
```

Goreq使用`goreq.Get`来创建请求，之后可以使用*链式操作*进行参数、请求头等的配置。最后，加上`.Do()`这个请求就会被`net/http`执行，得到返回结果。

```go
resp := goreq.Get("https://httpbin.org/get").AddParam("A","a").Do()
```

得到的`resp`是`*goreq.Response`，包含了相应的结果。Goreq会自动处理编码。

获取响应内容：

```go
fmt.Println(resp.Txt())
```

此外：

* `resp.Resp() (*Response, error)` 获取响应本身以及网络请求错误。
* `resp.Txt() (string, error)` 自动处理完编码并解析为文本后的内容以及网络请求错误。
* `resp.HTML() (*goquery.Document, error)`
* `resp.XML() (*xmlpath.Node, error)`
* `resp.BindXML(i interface{}) error`
* `resp.JSON() (gjson.Result, error)`
* `resp.BindJSON(i interface{}) error`
* `resp.Error() error` 网络请求错误。（正常情况下为`nil`）

其中`.AddParam("A","a")`是配置请求的*链式操作*，在Goreq中还有很多可用的配置函数。

Goreq可以设置中间件、更换Http Client。请见[Goreq](./goreq.md)一章。

## ⚡建立爬虫

```shell
go get -u github.com/zhshch2002/gospider
```

第一个例子：

```go
package main

import (
	"github.com/zhshch2002/goreq"
	"github.com/zhshch2002/gospider"
)

func main() {
    s := gospider.NewSpider() // 创建蜘蛛
    
    // 收到响应时
	s.OnResp(func(ctx *gospider.Context) {
		ctx.Println("OnResp")
	})
    
    s.OnItem(func(ctx *gospider.Context, i interface{}) interface{} { // 收集并存储结果
        ctx.Println(i)
        return i
    })

    s.SeedTask( // 种子任务
        goreq.Get("https://httpbin.org/get"),
        func(ctx *gospider.Context) {
            ctx.AddItem(ctx.Resp.Text) // 提交任务爬取结果
        },
    )

    s.Wait() // 等待所有任务完成并释放资源
}
```

Gospider的结构十分清晰。`s := gospider.NewSpider()`创建了一个蜘蛛，此后的操作都围绕这个蜘蛛进行。

`s.OnResp()`设置当收到响应时的回调函数，此外还有`OnTask`执行新任务前等诸多回调Hook。

`s.OnItem()`设置收集结果的函数。这一点类似scrapy的Pipeline。一个任务执行中，可以向任务的`*gospider.Context`添加任务结果，在所有回调执行完的情况下，蜘蛛会调用这个些函数来收集结果，进行数据库存储、文件存储等工作。

`s.SeedTask()`此时是添加的蜘蛛第一个任务。一般的任务需要调用`Context`的`ctx.AddTask()`创建，因为最初没有第一个任务，所以称之为SeedTask。调用`s.SeedTask()`将使用一个空的`Context`。

`func(ctx *gospider.Context)`这是`s.SeedTask()`的一个参数（`ctx.AddTask()`也相同）。是作为这个请求的处理函数。这一点与`scrapy`相似。

### `ctx *gospider.Context`

```go
type Context struct {
	Req   *goreq.Request
	Resp  *goreq.Response
	Meta  map[string]interface{}
}
```

`Context`包括任务的请求、响应、上一个任务传来的参数（`Meta map[string]interface{}`）。

`Req`和`Resp`参考[Goreq](./goreq.md)一章使用。

`Meta`参数随着调用`ctx.AddTask()`将自动传递到下一个任务里。`SeedTask`创建的任务`Meta`与`Req`为空

调用`ctx.Abort()`将中断任务的回调处理链，之后的回调函数，`OnResp`、`OnHTML`等将不会被执行。但回收结果的`OnItem`依旧会被执行。

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