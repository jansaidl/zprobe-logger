package main

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type worker struct {
	Index  string
	Runs   int64
	Cancel context.CancelFunc
}

//go:embed list.html
var list string
var listTemplate *template.Template

var mainCounter uint64
var mainLock sync.Mutex

func init() {
	listTemplate = template.Must(template.New("").Parse(list))
}

func (w *worker) stop() {
	w.Cancel()
}

func (w *worker) run(ctx context.Context) {
	cancelCtx, cancel := context.WithCancel(ctx)
	w.Cancel = cancel

	for {
		w.Runs++
		call()
		select {
		case <-cancelCtx.Done():
			return
		case <-time.After(time.Second * 1):
		}

	}
}

var workers []*worker

var inc = time.Now().UnixNano()

func GetRecordIdFromTimestamp(now time.Time) string {
	return fmt.Sprintf("%013s%013s", strconv.FormatInt(now.UnixNano(), 36), strconv.FormatInt(atomic.AddInt64(&inc, 1), 36))
}

func call() {
	mainLock.Lock()
	defer mainLock.Unlock()
	now := time.Now()

	mainCounter++
	fmt.Println(mainCounter, len(workers), GetRecordIdFromTimestamp(now), now.Format(time.RFC3339Nano))
}

func main() {
	log.Println("starting")
	ctx, cancel := context.WithCancel(context.Background())

	rand.Seed(time.Now().Unix())

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
		<-interrupt
		cancel()
	}()

	go func() {
		if err := http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			action := r.URL.Query().Get("do")
			switch action {
			case "echo":
				fmt.Fprintf(w, "%d", mainCounter)
				return

			case "add":
				addWorker(ctx)
				return
			case "remove":
				index := r.URL.Query().Get("index")
				for heyIndex, heyWorker := range workers {
					if heyWorker.Index == index {
						workers = append(workers[0:heyIndex], workers[heyIndex+1:]...)
						heyWorker.stop()
						fmt.Println("remove index: ", index)
						break
					}
				}
			}
			listTemplate.Execute(w, workers)
			return
		})); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

}

func addWorker(ctx context.Context) {
	w := newWorker()
	workers = append(workers, w)
	go w.run(ctx)
}

func newWorker() *worker {
	return &worker{
		Index: strconv.Itoa(rand.Int()),
	}
}
