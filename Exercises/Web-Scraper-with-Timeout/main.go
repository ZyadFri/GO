package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

func fetchURL(ctx context.Context, url string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", url, err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			fmt.Printf("Fetch for %s canceled or timed out: %v\n", url, ctx.Err())
			return
		default:
			fmt.Printf("Error fetching %s: %v\n", url, err)
			return
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response from %s: %v\n", url, err)
		return
	}

	fmt.Printf("Fetched %s: %d bytes\n", url, len(body))
}

func main() {
	urls := []string{
		"http://example.com",
		"http://httpbin.org/delay/2",
		"http://httpbin.org/delay/5",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			fetchURL(ctx, url)
		}(url)
	}

	wg.Wait()
	fmt.Println("All fetches complete.")
}