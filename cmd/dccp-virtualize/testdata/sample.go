package main

func main() {
	// Comment1
	go helloWorld()

	// Comment2
	go func() {
		fooBar()
	}()
}
