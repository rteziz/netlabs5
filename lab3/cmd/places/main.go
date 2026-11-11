package main

import (
	"fmt"
	"os"
)

// main запускает приложение и сообщает об ошибке завершения.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
