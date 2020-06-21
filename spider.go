package gospider

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/panjf2000/ants/v2"
	"github.com/tidwall/gjson"
	"github.com/zhshch2002/goreq"
	"log"
	"runtime"
	"time"
)

var (
	UnknownExt = errors.New("unknown ext")
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

	Client    *goreq.Client
	Status    *SpiderStatus
	Scheduler Scheduler
	taskPool  *ants.Pool
	itemPool  *ants.Pool

	onTaskHandlers      []func(ctx *Context, t *Task) *Task
	onRespHandlers      []Handler
	onItemHandlers      []func(ctx *Context, i interface{}) interface{}
	onRecoverHandlers   []func(ctx *Context, err error)
	onReqErrorHandlers  []func(ctx *Context, err error)
	onRespErrorHandlers []func(ctx *Context, err error)
}

func NewSpider(e ...interface{}) *Spider {
	pt, err := ants.NewPool(10)
	if err != nil {
		panic(err)
	}
	pi, err := ants.NewPool(10)
	if err != nil {
		panic(err)
	}
	s := &Spider{
		Name:      "gospider",
		Output:    true,
		Client:    goreq.NewClient(),
		Scheduler: NewBaseScheduler(false),
		Status:    NewSpiderStatus(),
		taskPool:  pt,
		itemPool:  pi,
	}
	s.Use(e...)
	s.schedule()
	return s
}

func (s *Spider) Use(exts ...interface{}) {
	for _, fn := range exts {
		switch fn.(type) {
		case func(*Spider):
			fn.(Extension)(s)
			break
		case goreq.Middleware:
			s.Client.Use(fn.(goreq.Middleware))
			break
		default:
			panic(UnknownExt)
		}
	}
}

func (s *Spider) Forever() {
	select {}
}

func (s *Spider) Wait() {
	time.Sleep(500 * time.Millisecond)
	for true {
		if s.taskPool.Running() == 0 && s.itemPool.Running() == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func (s *Spider) SetTaskPoolSize(i int) {
	s.taskPool.Tune(i)
}

func (s *Spider) SetItemPoolSize(i int) {
	s.itemPool.Tune(i)
}

func (s *Spider) schedule() {
	go func() {
		for true {
			if t := s.Scheduler.GetTask(); t != nil {
				err := s.taskPool.Submit(func() {
					s.handleTask(t)
				})
				if err != nil {
					panic(err)
				}
			} else {
				time.Sleep(500 * time.Millisecond)
			}
			runtime.Gosched()
		}
	}()
	go func() {
		for true {
			if i := s.Scheduler.GetItem(); i != nil {
				err := s.itemPool.Submit(func() {
					s.handleOnItem(i)
				})
				if err != nil {
					panic(err)
				}
			} else {
				time.Sleep(500 * time.Millisecond)
			}
			runtime.Gosched()
		}
	}()
}

func (s *Spider) handleTask(t *Task) {
	s.Status.FinishTask()
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
				log.Println("["+s.Name+"]", "recover from panic:", err, "ctx:", ctx, "\n", SprintStack())
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
			fmt.Println("["+s.Name+"]", "resp error:", ctx.Resp.Err, "ctx:", ctx)
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

func (s *Spider) addTask(t *Task) { //TODO
	s.Scheduler.AddTask(t)
	s.Status.AddTask()
}

func (s *Spider) addItem(i *Item) {
	s.Scheduler.AddItem(i)
	s.Status.AddItem()
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
				fmt.Println("["+s.Name+"]", "recover from panic:", err, "ctx:", i.Ctx, "\n", SprintStack())
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
