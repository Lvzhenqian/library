package fn

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWithContext(t *testing.T) {
	g, _ := WithContext(context.Background())
	for i := 0; i < 5; i++ {
		i := i
		g.Go(func() error {
			time.Sleep(time.Second)
			if i%2 != 0 {
				return fmt.Errorf("some error Id: %d", i)
			}
			return nil
		})
	}

	for _, err := range g.Wait() {
		t.Log(err)
	}
}
