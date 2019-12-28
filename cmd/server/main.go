package main

import (
	gif "github.com/belljustin/gif/pkg"
)

func main() {
	r := gif.NewRouter()
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
