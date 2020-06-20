package gospider

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

type SpiderStatus struct {
	TotalTask    int64
	FinishedTask int64
	TotalItem    int64
	ExecSpeed    int64
	itemSpeed    int64
}

func NewSpiderStatus() *SpiderStatus {
	s := &SpiderStatus{}
	lastFinish := int64(0)
	lastItem := int64(0)
	go func() {
		for true {
			s.ExecSpeed = (s.FinishedTask - lastFinish) / 5
			s.itemSpeed = (s.TotalItem - lastItem) / 5
			lastFinish = s.FinishedTask
			lastItem = s.TotalItem
			time.Sleep(5 * time.Second)
		}
	}()
	return s
}

func (s *SpiderStatus) AddTask() {
	atomic.AddInt64(&s.TotalTask, 1)
}

func (s *SpiderStatus) AddItem() {
	atomic.AddInt64(&s.TotalTask, 1)
}

func (s *SpiderStatus) FinishTask() {
	atomic.AddInt64(&s.FinishedTask, 1)
}

func (s *SpiderStatus) PrintSignalLine(name string) {
	log.Println(
		"["+name+"]",
		s.TotalItem,
		"items("+fmt.Sprint(s.itemSpeed)+"item/sec)",
		s.FinishedTask,
		"task finished("+fmt.Sprint(s.ExecSpeed)+"task/sec)",
		s.TotalTask, "task total"+fmt.Sprint(int((float64(s.FinishedTask)/float64(s.TotalTask))*100))+"%",
	)
}
