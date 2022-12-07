package groupsync

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestGroup_Go(t *testing.T) {
	data := make([]int, 0)
	g, err := NewGroup(&data, WithLimit(50), WithChannelBuffer(10))
	if err != nil {
		t.Fatal(err)
	}
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		i := i
		g.Go(func() int {
			tt := rand.Intn(500)
			time.Sleep(time.Millisecond * time.Duration(tt))
			fmt.Printf("now: %d,sleep: %d\n", i, tt)
			return i
		})
	}

	g.Wait()

	t.Log(data)
}
