package runner

import (
	"context"
	"crashlens/db"
	"errors"
	"log"
	"sync"
	"time"
)

var ErrQueueFull = errors.New("job queue is full")
var ErrStopped = errors.New("job runner is stopped")

type queuedJob struct {
	id   int
	path string
}
type Manager struct {
	database *db.DB
	queue    chan queuedJob
	ctx      context.Context
	cancel   context.CancelFunc
	timeout  time.Duration
	workers  sync.WaitGroup
	mu       sync.Mutex
	stopped  bool
}

func NewManager(database *db.DB, concurrency, queueSize int, timeout time.Duration) *Manager {
	if concurrency < 1 || queueSize < 1 || timeout <= 0 {
		panic("invalid runner limits")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{database: database, queue: make(chan queuedJob, queueSize), ctx: ctx, cancel: cancel, timeout: timeout}
	for i := 0; i < concurrency; i++ {
		m.workers.Add(1)
		go m.work()
	}
	return m
}
func (m *Manager) Submit(name, kind, path string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return 0, ErrStopped
	}
	if len(m.queue) == cap(m.queue) {
		return 0, ErrQueueFull
	}
	id, err := m.database.CreateManagedWorkload(name, kind)
	if err != nil {
		return 0, err
	}
	m.queue <- queuedJob{int(id), path}
	return id, nil
}
func (m *Manager) work() {
	defer m.workers.Done()
	for {
		select {
		case <-m.ctx.Done():
			return
		case job := <-m.queue:
			ctx, cancel := context.WithTimeout(m.ctx, m.timeout)
			if _, err := RunPythonJobContext(ctx, job.path, job.id, m.database); err != nil {
				log.Printf("Job %d: %v", job.id, err)
			}
			cancel()
		}
	}
}
func (m *Manager) Close() error {
	m.mu.Lock()
	m.stopped = true
	m.cancel()
	m.mu.Unlock()
	m.workers.Wait()
	return m.database.RecoverManagedJobs("backend stopped before completion")
}
