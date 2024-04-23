# go-queue

`go-queue` is a small lightweight package for handling a queue in a go application. It features configurable number of workers, and you can provide your own logging or it can just use `os.Stdout`

## Examples

Simple example of instantiating and using the queue
```go
import (
	"context"
	"fmt"
	"os"

	"github.com/codemonkey76/go-queue"
)

type Task struct {
	queue.BaseTask
}

func (t *Task) Handle(ctx context.Context) error {
	fmt.Println("Hello World")
	return nil
}

func (t *Task) ID() string {
	return "PrintHello"
}

func main() {
	task := &Task{}
	queue := queue.NewWorkerQueue(10, 10, 10, 10, os.Stdout)
	defer queue.Shutdown()

	queue.Enqueue(task)
}
```

## License

Copyright (c) 2015-present [Shane Poppleton](https://github.com/codemonkey76)

Licensed under [MIT License](https://github.com/codemonkey76/go-queue/master/LICENSE)
