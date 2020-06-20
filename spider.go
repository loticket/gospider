package gospider

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/panjf2000/ants/v2"
	"github.com/tidwall/gjson"
	"github.com/zhshch2002/goreq"
	"log"
	"time"
)

type Handler func(ctx *Context)
type Extension func(s *Spider)

type Task struct {
	Req      *goreq.Request
	Handlers []Handler
	Meta     map[string]interface{}
}

type Item struct {
	Ctx  *Context
	Data interface{}
}

func NewTask(req *goreq.Request, meta map[string]interface{}, a ...Handler) (t *Task) {
	t = &Task{
		Req:      req,
		Handlers: a,
		Meta:     meta,
	}
	return
}

type Spider struct {
	Name   string
	Output bool

	Client   *goreq.Client
	status   *SpiderStatus
	taskPool *ants.Pool
	itemPool *ants.Pool

	onTaskHandlers      []func(ctx *Context, t *Task) *Task
	onRespHandlers      []Handler
	onItemHandlers      []func(ctx *Context, i interface{}) interface{}
	onRecoverHandlers   []func(ctx *Context, err error)
	onReqErrorHandlers  []func(ctx *Context, err error)
	onRespErrorHandlers []func(ctx *Context, err error)
}

func NewSpider(e ...Extension) *Spider {
	pt, err := ants.NewPool(10)
	if err != nil {
		panic(err)
	}
	pi, err := ants.NewPool(10)
	if err != nil {
		panic(err)
	}
	s := &Spider{
		Name:     "gospider",
		Output:   true,
		Client:   goreq.NewClient(),
		status:   NewSpiderStatus(),
		taskPool: pt,
		itemPool: pi,
	}
	for _, fn := range e {
		fn(s)
	}
	return s
}

func (s *Spider) Forever() {
	select {}
}

func (s *Spider) Wait() {
	for true {
		if s.taskPool.Running() == 0 && s.itemPool.Running() == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func (s *Spider) handleTask(t *Task) {
	s.status.FinishTask()
	ctx := &Context{
		s:     s,
		Req:   t.Req,
		Resp:  nil,
		Meta:  t.Meta,
		abort: false,
	}
	defer func() {
		if err := recover(); err != nil {
			if s.Output {
				log.Println("["+s.Name+"]", "recover from panic:", err, "ctx:", ctx)
			}
			if e, ok := err.(error); ok {
				s.handleOnError(ctx, e)
			} else {
				s.handleOnError(ctx, fmt.Errorf("%v", err))
			}
		}
	}()
	if t.Req.Err != nil {
		if s.Output {
			fmt.Println("["+s.Name+"]", "req config error:", t.Req.Err, "ctx:", ctx)
		}
		s.handleOnReqError(ctx, t.Req.Err)
		return
	}
	ctx.Resp = s.Client.Do(t.Req)
	if ctx.Resp.Err != nil {
		if s.Output {
			fmt.Println("["+s.Name+"]", "resp error:", t.Req.Err, "ctx:", ctx)
		}
		s.handleOnRespError(ctx, ctx.Resp.Err)
		return
	}
	if s.Output {
		fmt.Println("["+s.Name+"]", ctx)
	}
	s.handleOnResp(ctx)
	if ctx.IsAborted() {
		return
	}
	for _, fn := range t.Handlers {
		fn(ctx)
		if ctx.IsAborted() {
			return
		}
	}
}

func (s *Spider) SeedTask(req *goreq.Request, h ...Handler) {
	ctx := &Context{
		s:     s,
		Req:   nil,
		Resp:  nil,
		Meta:  map[string]interface{}{},
		abort: false,
	}
	ctx.AddTask(req, h...)
}

func (s *Spider) addTask(t *Task) {
	err := s.taskPool.Submit(func() {
		s.handleTask(t)
	})
	if err != nil {
		panic(err)
	}
	s.status.AddTask()
}

func (s *Spider) addItem(i *Item) {
	err := s.itemPool.Submit(func() {
		s.handleOnItem(i)
	})
	if err != nil {
		panic(err)
	}
	s.status.AddItem()
}

/*************************************************************************************/
func (s *Spider) OnTask(fn func(ctx *Context, t *Task) *Task) {
	s.onTaskHandlers = append(s.onTaskHandlers, fn)
}
func (s *Spider) handleOnTask(ctx *Context, t *Task) *Task {
	for _, fn := range s.onTaskHandlers {
		t = fn(ctx, t)
		if t == nil {
			return t
		}
	}
	return t
}

/*************************************************************************************/
func (s *Spider) OnResp(fn Handler) {
	s.onRespHandlers = append(s.onRespHandlers, fn)
}
func (s *Spider) OnHTML(selector string, fn func(ctx *Context, sel *goquery.Selection)) {
	s.OnResp(func(ctx *Context) {
		if ctx.Resp.IsHTML() {
			if h, err := ctx.Resp.HTML(); err == nil {
				h.Find(selector).Each(func(i int, selection *goquery.Selection) {
					fn(ctx, selection)
				})
			}
		}
	})
}
func (s *Spider) OnJSON(q string, fn func(ctx *Context, j gjson.Result)) {
	s.onRespHandlers = append(s.onRespHandlers, func(ctx *Context) {
		if ctx.Resp.IsJSON() {
			if j, err := ctx.Resp.JSON(); err == nil {
				if res := j.Get(q); res.Exists() {
					fn(ctx, res)
				}
			}
		}
	})
}
func (s *Spider) handleOnResp(ctx *Context) {
	for _, fn := range s.onRespHandlers {
		if ctx.IsAborted() {
			return
		}
		fn(ctx)
	}
}

/*************************************************************************************/
func (s *Spider) OnItem(fn func(ctx *Context, i interface{}) interface{}) {
	s.onItemHandlers = append(s.onItemHandlers, fn)
}
func (s *Spider) handleOnItem(i *Item) {
	defer func() {
		if err := recover(); err != nil {
			if s.Output {
				fmt.Println("["+s.Name+"]", "recover from panic:", err, "ctx:", i.Ctx)
			}
			if e, ok := err.(error); ok {
				s.handleOnError(i.Ctx, e)
			} else {
				s.handleOnError(i.Ctx, fmt.Errorf("%v", err))
			}
		}
	}()
	for _, fn := range s.onItemHandlers {
		i.Data = fn(i.Ctx, i.Data)
		if i.Data == nil {
			return
		}
	}
}

/*************************************************************************************/
func (s *Spider) OnRecover(fn func(ctx *Context, err error)) {
	s.onRecoverHandlers = append(s.onRecoverHandlers, fn)
}
func (s *Spider) handleOnError(ctx *Context, err error) {
	for _, fn := range s.onRecoverHandlers {
		fn(ctx, err)
	}
}
func (s *Spider) OnRespError(fn func(ctx *Context, err error)) {
	s.onRespErrorHandlers = append(s.onRespErrorHandlers, fn)
}
func (s *Spider) handleOnRespError(ctx *Context, err error) {
	for _, fn := range s.onRespErrorHandlers {
		fn(ctx, err)
	}
}
func (s *Spider) OnReqError(fn func(ctx *Context, err error)) {
	s.onReqErrorHandlers = append(s.onReqErrorHandlers, fn)
}
func (s *Spider) handleOnReqError(ctx *Context, err error) {
	for _, fn := range s.onReqErrorHandlers {
		fn(ctx, err)
	}
}
