package gospider

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/loticket/goreq"
)

type Context struct {
	s     *Spider
	Req   *goreq.Request
	Resp  *goreq.Response
	Meta  map[string]interface{}
	abort bool
}

// Abort this context to break the handler chain and stop handling
func (c *Context) Abort() {
	c.abort = true
}

// IsAborted return was the context dropped
func (c *Context) IsAborted() bool {
	return c.abort
}

// addTask add a task to new task list. After every handler func return,spider will collect these tasks
func (c *Context) AddTask(req *goreq.Request, h ...Handler) {
	if !req.URL.IsAbs() {
		req.URL = c.Req.URL.ResolveReference(req.URL)
	}
	t := c.s.handleOnTask(c, NewTask(req, c.Meta, h...))
	if t == nil {
		return
	}
	c.s.addTask(t)
}

// addItem add an item to new item list. After every handler func return,
// spider will collect these items and call OnItem handler func
func (c *Context) AddItem(i interface{}) {
	c.s.addItem(&Item{
		Ctx:  c,
		Data: i,
	})
}

func (c *Context) IsDownloaded() bool {
	return c.Resp != nil
}

func (c *Context) Println(v ...interface{}) {
	log.Print(v...)
}

func (c *Context) Error() *zerolog.Event {
	return log.Error()
}

func (c *Context) String() string {
	if c.Req == nil {
		return "[empty context]"
	} else if c.Resp == nil {
		return fmt.Sprint("[not downloaded ctx] ", c.Req.URL.String())
	} else if c.Resp.Response == nil || c.Resp.Err != nil {
		return fmt.Sprint("[err ctx] ", c.Req.URL.String())
	} else {
		return fmt.Sprint("["+c.Resp.Status+"] ", c.Req.URL)
	}
}
