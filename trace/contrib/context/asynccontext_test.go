package context

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCancelCtx(t *testing.T) {
	go func() {
		fmt.Println("parent goroutine start")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		asyncWork(ctx)
		time.Sleep(1 * time.Second)
		fmt.Println("parent goroutine exit")
	}()

	time.Sleep(20 * time.Second)
}

func TestAsyncCtx(t *testing.T) {
	go func() {
		fmt.Println("parent goroutine start")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		asyncWork(NewContextForAsyncTracing(ctx))
		time.Sleep(1 * time.Second)
		fmt.Println("parent goroutine exit")
	}()

	time.Sleep(20 * time.Second)
}

func asyncWork(ctx context.Context) {
	go func() {
		select {
		case <-time.After(3 * time.Second):
			fmt.Println("child goroutine work done. goroutine close")
		case <-ctx.Done():
			fmt.Println("child goroutine ctx canceled")
		}
	}()
}
