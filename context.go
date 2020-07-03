package gospider

import (
	"fmt"
	"github.com/zhshch2002/goreq"
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

func (c *Context) Printf(format string, v ...interface{}) {
	log.Printf("%v "+format, append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
}
func (c *Context) Print(v ...interface{}) {
	log.Print(append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
}
func (c *Context) Println(v ...interface{}) {
	log.Println(append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
}
func (c *Context) Fatalf(format string, v ...interface{}) {
	log.Fatalf("%v "+format, append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
}
func (c *Context) Fatal(v ...interface{}) {
	log.Fatal(append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
}
func (c *Context) Fatalln(v ...interface{}) {
	log.Fatalln(append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
}
func (c *Context) Panicf(format string, v ...interface{}) {
	log.Panicf("%v "+format, append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
}
func (c *Context) Panic(v ...interface{}) {
	log.Panic(append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
}
func (c *Context) Panicln(v ...interface{}) {
	log.Panicln(append([]interface{}{"[" + c.s.Name + "]"}, v...)...)
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
