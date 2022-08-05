package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Data struct {
	Items []Item
}

type Item struct {
	Id      string
	Content string
}

var inc = time.Now().UnixNano()

func GetRecordIdFromTimestamp(now time.Time) string {
	return fmt.Sprintf("%013s%013s", strconv.FormatInt(now.UnixNano(), 36), strconv.FormatInt(atomic.AddInt64(&inc, 1), 36))
}

func main() {

	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	from := "0cly585861hxk0cly548uqfvdn"

	u, err := url.Parse("wss://logs.app.zerops.dev/api/rest/log/stream?accessToken=zom2Gig9QnAB3zLWNKYTEw&containerId=QN85uSBcTQ2JckvUOBx9Pw&from=" + from)
	if err != nil {
		panic(err)
	}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	r := regexp.MustCompile("^[0-9]+ [0-9]+ [0-9a-zA-Z]+ ")

	go func() {
		defer close(done)
		lastSeq := 0
		for {
			var d Data
			_, message, err := c.ReadMessage()
			if err != nil {
				panic(err)
				return
			}
			if err := json.Unmarshal(message, &d); err != nil {
				panic(err)
			}
			for _, i := range d.Items {
				if !r.MatchString(i.Content) {
					fmt.Println("SKIP", i.Content)
					continue
				}
				parts := strings.Split(i.Content, " ")
				if len(parts) < 4 {
					fmt.Println(i.Id, i.Content)
					panic("invalid count")
				}
				newSeq, err := strconv.Atoi(parts[0])
				if err != nil {
					fmt.Println(i.Id, i.Content)
					panic(err)
				}
				if lastSeq+1 != newSeq {
					fmt.Println(i.Id, i.Content)
					fmt.Printf("expected %d, have %d\n", lastSeq, newSeq)
				}
				lastSeq = newSeq
				if lastSeq%1000 == 0 {
					fmt.Println(i.Content)
				}
			}
		}
	}()

	<-interrupt

}
