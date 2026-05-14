package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fireharp/tinkershop/internal/cli"
)

func main() {
	args := append([]string{"daemon"}, os.Args[1:]...)
	if err := cli.Run(context.Background(), args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
