package queue

import (
	"context"
	"io"
	"log"
	"math"
	"sync"
	"time"
)

type WorkerQueue struct {
	tasks           chan Queueable
	workers         int
	initialWorkers  int
	maxWorkers      int
	shutdownSignal  chan struct{}
	workerIncrement int
	cancelFuncs     []context.CancelFunc
	mutex           sync.Mutex
	wg              sync.WaitGroup
	logger          *log.Logger
}

func (wq *WorkerQueue) Enqueue(task Queueable) {
	wq.logger.Printf("Enqueuing task %s\n", task.ID())
	wq.tasks <- task
}

func NewWorkerQueue(initialWorkers, maxWorkers, queueSize, workerIncrement int, logWriter io.Writer) *WorkerQueue {
	logger := log.New(logWriter, "queue: ", log.Ldate|log.Ltime|log.Lshortfile)

	if initialWorkers > maxWorkers {
		initialWorkers = maxWorkers
	}

	if workerIncrement < 1 {
		workerIncrement = 1
	}

	wq := &WorkerQueue{
		tasks:           make(chan Queueable, queueSize),
		workers:         initialWorkers,
		initialWorkers:  initialWorkers,
		maxWorkers:      maxWorkers,
		shutdownSignal:  make(chan struct{}),
		workerIncrement: workerIncrement,
		logger:          logger,
	}

	wq.startWorkers()
	wq.startAdjustingWorkers(10 * time.Second)
	return wq
}

func (wq *WorkerQueue) startWorkers() {
	for i := 0; i < wq.workers; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		wq.cancelFuncs = append(wq.cancelFuncs, cancel)
		wg.Add(1)
		go wq.worker(ctx)
	}
}

func (wq *WorkerQueue) worker(ctx context.Context) {
	defer wq.wg.Done()
	for {
		select {
		case task, ok := <-wq.tasks:
			if !ok {
				return
			}
			taskCtx, taskCancel := context.WithTimeout(ctx, task.Timeout())
			err := task.Handle(taskCtx)
			wq.logger.Printf("Task %s completed\n", task.ID())
			taskCancel()

			if err != nil && task.ShouldRetry() {
				wq.logger.Printf("Task failed, adding to retry queue: %s\n", err.Error())
				task.Retry()
				delay := time.Duration(math.Pow(2, float64(task.RetryCount()))) * time.Second
				time.AfterFunc(delay, func() {
					wq.Enqueue(task)
				})
			}
		case <-ctx.Done():
			return
		}
	}
}

func (wq *WorkerQueue) startAdjustingWorkers(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				wq.adjustWorkers()
			case <-wq.shutdownSignal:
				ticker.Stop()
				return
			}
		}
	}()
}

func (wq *WorkerQueue) adjustWorkers() {
	currentLength := len(wq.tasks)
	currentCapacity := cap(wq.tasks)

	if currentLength >= currentCapacity*80/100 && wq.workers < wq.maxWorkers { // 80% full
		wq.logger.Println("Queue is >80% Full, scaling up")
		wq.scaleUp()
	} else if currentLength <= currentCapacity*20/100 && wq.workers > wq.initialWorkers { // 20% full
		wq.logger.Println("Queue is <20% Full, scaling down")
		wq.scaleDown()
	}
}

func (wq *WorkerQueue) scaleUp() {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()

	for i := 0; i < wq.workerIncrement && wq.workers < wq.maxWorkers; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		wq.cancelFuncs = append(wq.cancelFuncs, cancel)
		wq.wg.Add(1)
		go wq.worker(ctx)
		wq.workers++
	}
}

func (wq *WorkerQueue) scaleDown() {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()

	numWorkersToStop := min(wq.workerIncrement, wq.workers-wq.initialWorkers)
	for i := 0; i < numWorkersToStop; i++ {
		cancelFunc := wq.cancelFuncs[len(wq.cancelFuncs)-1]
		cancelFunc()
		wq.cancelFuncs = wq.cancelFuncs[:len(wq.cancelFuncs)-1]
		wq.workers--
	}
}

func (wq *WorkerQueue) Shutdown() {
	close(wq.tasks)
	wq.wg.Wait()
}
