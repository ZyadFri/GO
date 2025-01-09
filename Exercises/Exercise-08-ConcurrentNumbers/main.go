package main

import (
	"fmt"
	"sync"
)

func squareNumber(number int, wg *sync.WaitGroup) {
	defer wg.Done()
	result := number * number
	fmt.Printf("Square of %d is %d\n", number, result)
}

func main() {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	var wg sync.WaitGroup

	for _, number := range numbers {
		wg.Add(1)
		go squareNumber(number, &wg)
	}

	fmt.Println("All goroutines completed!")
}