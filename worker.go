package himawari

import (
	"log"
	"sync"
	"time"
)

type Work interface {
	Name() string
	Do() error
	MaxFailTimes() int
}

type Worker struct {
	work       Work
	TriedTimes int
	Done       bool
	manager    *Manager
}

func NewWorker(work Work, manager *Manager) *Worker {
	return &Worker{
		work:    work,
		manager: manager,
	}
}

func (w *Worker) Work() {
	for {
		if w.TriedTimes >= w.work.MaxFailTimes() {
			w.Done = false
			break
		}
		start := time.Now()
		err := w.work.Do()
		dur := time.Since(start)
		if err == nil {
			log.Printf("[%s] tried: %d use: %v done",
				w.work.Name(), w.TriedTimes, dur)
			w.Done = true
			break
		}
		log.Printf("[%s] tried: %d use: %v err: %v",
			w.work.Name(), w.TriedTimes, dur, err)
		w.TriedTimes++
	}
	w.manager.wg.Done()
}

type Manager struct {
	workers []*Worker
	wg      *sync.WaitGroup
}

func NewManager() *Manager {
	return &Manager{
		wg: &sync.WaitGroup{},
	}
}

func (m *Manager) NewWork(work Work) {
	worker := NewWorker(work, m)
	m.workers = append(m.workers, worker)
	m.wg.Add(1)
	go worker.Work()
}

func (m *Manager) Done() bool {
	m.wg.Wait()
	for _, worker := range m.workers {
		if !worker.Done {
			return false
		}
	}
	return true
}
