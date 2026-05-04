package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
)

func testFunc() int {
	return 123
}
func testFunc2() {
	fmt.Println("called testfunc2")
}
func DoWork(i int) int {
	fmt.Printf("%v", i)
	return i
}
func oldWg() {
	dataChan := make(chan int)
	testFunc()
	testFunc2()
	go func() {
		wg := sync.WaitGroup{}
		for i := range 1000 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				dataChan <- DoWork(i)
			}()
		}
		wg.Wait()
		close(dataChan)
	}()
	for i := range dataChan {
		fmt.Printf("datachan i: %v\n", i)
	}
}
func newWg() {
	dataChan := make(chan int)
	var (
		wg sync.WaitGroup
	)

	go func() {
		for i := range 1000 {
			wg.Go(func() {
				dataChan <- DoWork(i)
			})
		}
		wg.Wait()
		close(dataChan)
	}()

	for i := range dataChan {
		fmt.Printf("datachan i: %v\n", i)
	}
}

func main() {
	fmt.Println("scratch")
	// oldWg()

	newWg()
	var (
		reader io.Reader
	)
	req, err := http.NewRequestWithContext(context.Background(), "GET", "https://www.google.com", reader)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("err: %v", err)
	}
	fmt.Printf("res: %v", res)
	fmt.Println("break")
}
