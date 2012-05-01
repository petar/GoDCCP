package main

func main() {
	// Comment1
	go helloWorld()

	// Comment2
	go func() {
		fooBar()
		// This call will be rewritten too
		time.Now()
		time.Sleep(1e9)
	}()

	x, ok := <-ch

	ch <- y
}
