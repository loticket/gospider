package main

import (
	"fmt"
	"github.com/zhshch2002/goreq"
	"github.com/zhshch2002/gospider"
)

func main() {
	s := gospider.NewSpider()
	s.SeedTask(
		goreq.Get("https://httpbin.org/get"),
		func(ctx *gospider.Context) {
			ctx.AddItem(ctx.Resp.Text)
		},
	)
	s.OnItem(func(ctx *gospider.Context, i interface{}) interface{} {
		fmt.Println(i)
		panic("aaaaa")
		return i
	})
	s.Wait()
}
