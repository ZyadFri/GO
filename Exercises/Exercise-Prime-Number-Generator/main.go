package main

import (
	"fmt"
)

func generatePrimes(n int, ch chan<- int) {
	for i := 2; i <= n; i++ {
		if isPrime(i) {
			ch <- i
		}
	}
	close(ch)
}

func isPrime(num int) bool {
	if num < 2 {
		return false
	}
	for i := 2; i*i <= num; i++ {
		if num%i == 0 {
			return false
		}
	}
	return true
}

func printPrimes(ch <-chan int) {
	for prime := range ch {
		fmt.Println(prime)
	}
}

func main() {

	primes := make(chan int)

	go generatePrimes(50, primes)

	printPrimes(primes)
}
