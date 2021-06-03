package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Commits struct {
	Nr      int
	Commits []string
}

var (
	Counter      = make(map[int]*Commits)
	TotalCommits uint64
)

func parseFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) == 0 && err != nil {
			break
		}

		TotalCommits += 1
		line = strings.TrimRight(line, "\n")
		items := strings.SplitN(line, " ", 2)
		if len(items) != 2 && len(items[0]) != 40 {
			fmt.Printf("fail to parse line: %s\n", line)
		}
		width := len(items[1])
		if _, ok := Counter[width]; !ok {
			Counter[width] = &Commits{Nr: 0, Commits: []string{}}
		}
		c := Counter[width]
		c.Nr += 1
		if c.Nr < 10 {
			c.Commits = append(c.Commits, items[0])
		}

		if err != nil {
			break
		}
	}
	return nil
}

func showStat() {
	var (
		allWidth        = []int{}
		num      uint64 = 0
	)

	for width := range Counter {
		allWidth = append(allWidth, width)
	}
	sort.Ints(allWidth)
	for _, width := range allWidth {
		num += uint64(Counter[width].Nr)
		fmt.Printf("subject width %-3d: %5d commits, p %5.2f %%\n",
			width,
			Counter[width].Nr,
			100.0*float64(num)/float64(TotalCommits),
		)
		for idx := range Counter[width].Commits {
			if idx > 2 {
				break
			}
			fmt.Printf("\tcommit: %s\n", Counter[width].Commits[idx])
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: %s <filename>\n", os.Args[0])
		os.Exit(1)
	}
	parseFile(os.Args[1])
	showStat()
}
