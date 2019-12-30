package pkg

import (
	"bufio"
	"math/rand"
	"os"
	"time"
)

func LoadPrompts(fname string) ([]string, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var prompts []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		prompts = append(prompts, scanner.Text())
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(prompts), func(i, j int) {
		prompts[i], prompts[j] = prompts[j], prompts[i]
	})
	return prompts, scanner.Err()
}
