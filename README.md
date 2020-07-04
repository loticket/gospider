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