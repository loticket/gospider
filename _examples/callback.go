package main

import (
	"github.com/zhshch2002/goreq"
	"github.com/zhshch2002/gospider"
)

func main() {
	s := gospider.NewSpider()

	// 当新的任务被执行前（返回nil以取消任务）
	s.OnTask(func(ctx *gospider.Context, t *gospider.Task) *gospider.Task {
		ctx.Println("OnTask")
		return t
	})
	// 收到响应时
	s.OnResp(func(ctx *gospider.Context) {
		ctx.Println("OnResp")
	})
	// 处理通过`ctx.AddItem()`提交的结果（返回nil以中断多个回调函数连续处理），独立处理以减小对网络处理的阻塞
	s.OnItem(func(ctx *gospider.Context, i interface{}) interface{} {
		ctx.Println("OnItem", i)
		return i
	})
	// 在蜘蛛执行中出现panic
	s.OnRecover(func(ctx *gospider.Context, err error) {
		ctx.Println("OnRecover", err)
	})
	// 在创建新的requests时出现错误
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
		func(ctx *gospider.Context) { // 与此任务绑定的回调函数，等同于针对这个请求的OnResp。
			ctx.AddTask(goreq.Get("https://httpbin.org/get")) // 使用ctx创建的任务可以记录上一个请求的信息，再由其他扩展添加Referer等信息。
			ctx.AddItem(ctx.Resp.Text)
		},
	)
	s.SeedTask(goreq.Get("htps://httpbin.org/get"))
	s.Wait() // 等待所有任务执行完成并释放资源
}
