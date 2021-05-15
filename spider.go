package gospider

import (
	"errors"
	"fmt"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
	"github.com/loticket/goreq"
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
	Name    string
	Logging bool

	Client *goreq.Client
	Status *SpiderStatus
	wg     sync.WaitGroup

	onTaskHandlers      []func(ctx *Context, t *Task) *Task
	onRespHandlers      []Handler
	onItemHandlers      []func(ctx *Context, i interface{}) interface{}
	onRecoverHandlers   []func(ctx *Context, err error)
	onReqErrorHandlers  []func(ctx *Context, err error)
	onRespErrorHandlers []func(ctx *Context, err error)
}

func NewSpider(e ...interface{}) *Spider {
	s := &Spider{
		Name:    "spider",
		Logging: true,
		Client:  goreq.NewClient(),
		Status:  NewSpiderStatus(),
		wg:      sync.WaitGroup{},
	}
	s.Use(e...)
	return s
}

func (s *Spider) Use(exts ...interface{}) {
	for _, fn := range exts {
		switch fn.(type) {
		case func(s *Spider):
			fn.(func(s *Spider))(s)
			break
		case Extension:
			fn.(Extension)(s)
			break
		case goreq.Middleware, func(*goreq.Client, goreq.Handler) goreq.Handler:
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
			if s.Logging {
				log.Error().Err(fmt.Errorf("%v", err)).Str("spider", s.Name).Str("context", fmt.Sprint(ctx)).Str("stack", SprintStack()).Msg("handler recover from panic")
			}
			if e, ok := err.(error); ok {
				s.handleOnError(ctx, e)
			} else {
				s.handleOnError(ctx, fmt.Errorf("%v", err))
			}
		}
	}()
	if t.Req.Err != nil {
		if s.Logging {
			log.Error().Err(fmt.Errorf("%v", ctx.Req.Err)).Str("spider", s.Name).Str("context", fmt.Sprint(ctx)).Str("stack", SprintStack()).Msg("req error")
		}
		s.handleOnReqError(ctx, t.Req.Err)
		return
	}
	ctx.Resp = s.Client.Do(t.Req)
	if ctx.Resp.Err != nil {
		if s.Logging {
			log.Error().Err(fmt.Errorf("%v", ctx.Resp.Err)).Str("spider", s.Name).Str("context", fmt.Sprint(ctx)).Str("stack", SprintStack()).Msg("resp error")
		}
		s.handleOnRespError(ctx, ctx.Resp.Err)
		return
	}
	if s.Logging {
		log.Debug().Str("Spider", s.Name).Str("context", fmt.Sprint(ctx)).Msg("Finish")

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

func (s *Spider) SeedTask(req *goreq.Request, meta map[string]interface{}, h ...Handler) {
	ctx := &Context{
		s:     s,
		Req:   nil,
		Resp:  nil,
		Meta:  meta,
		abort: false,
	}
	ctx.AddTask(req, h...)
}

func (s *Spider) addTask(t *Task) {
	go func() {
		s.handleTask(t)
	}()
	s.Status.AddTask()
}

func (s *Spider) addItem(i *Item) {
	go func() {
		s.handleOnItem(i)
	}()
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
			if s.Logging {
				log.Error().Err(fmt.Errorf("%v", err)).Str("spider", s.Name).Str("context", fmt.Sprint(i.Ctx)).Str("stack", SprintStack()).Msg("OnItem recover from panic")
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
