package main

import (
	"flag"

	gif "github.com/belljustin/gif/pkg"
)

var fname string

func init() {
	flag.StringVar(&fname, "prompts", "/etc/gif/prompts.txt", "path to prompts.txt file")
}

func main() {
	flag.Parse()

	prompts, err := gif.LoadPrompts(fname)
	if err != nil {
		panic(err)
	}

	r := gif.NewRouter(prompts)
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
