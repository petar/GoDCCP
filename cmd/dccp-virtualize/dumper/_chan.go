package main

func main() {

	x, ok := <-ch

	ch <- y

	select {
	case <-ch:
	case ch <- 5:
	default:
	}
}
