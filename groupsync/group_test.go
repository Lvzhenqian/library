package groupsync

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestGroup_Go(t *testing.T) {

	g, err := NewSendGroup[int](10)
	if err != nil {
		t.Fatal(err)
	}
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		i := i
		g.Go(func(c chan<- int) {
			tt := rand.Intn(500)
			time.Sleep(time.Millisecond * time.Duration(tt))
			fmt.Printf("now: %d\n", i)
			c <- i
		})
	}

	t.Log(g.Wait())
}
