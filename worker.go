package main

import (
	"fmt"
	"sync"

	"github.com/gosuri/uiprogress"
)

type Pool struct {
	checks checks

	concurrency int
	checkChan   chan *check
	wg          sync.WaitGroup

	bar *uiprogress.Bar
}

func NewPool(checks checks, concurrency int) *Pool {
	return &Pool{
		checks:      checks,
		concurrency: concurrency,
		checkChan:   make(chan *check),
	}
}

func (p *Pool) Run(progress bool) {
	if progress {
		uiprogress.Start()
		p.bar = uiprogress.AddBar(len(p.checks))
		p.bar.AppendCompleted()
		p.bar.PrependElapsed()
		p.bar.PrependFunc(func(b *uiprogress.Bar) string {
			return fmt.Sprintf("Task (%d/%d)", b.Current(), len(p.checks))
		})
	}

	for i := 0; i < p.concurrency; i++ {
		go func() {
			p.work(i)
		}()
	}

	p.wg.Add(len(p.checks))
	for _, check := range p.checks {
		p.checkChan <- check
	}

	close(p.checkChan)

	p.wg.Wait()

	if progress {
		uiprogress.Stop()
	}
}

func (p *Pool) work(i int) {
	for check := range p.checkChan {
		check.run(&p.wg)
		if p.bar != nil {
			p.bar.Incr()
		}
	}
}
