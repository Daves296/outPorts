package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type portRange struct {
	min uint16
	max uint16
}

const (
	RED    = "\033[31m"
	GREEN  = "\033[32m"
	NORMAL = "\033[0m"
	BOLD   = "\033[1m"
)

var (
	successMsg, failureMsg     string
	printSuccess, printFailure = true, true
	wg                         = sync.WaitGroup{}
	d                          = net.Dialer{}
	out                        = make(chan string)
	exit                       = make(chan struct{})
)

func main() {
	log.SetPrefix("outPorts: ")
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Stderr.WriteString(`Examples
  check from ports 1 to 65535
        outPorts all
  check from ports 20-30 and then 40-50
        outPorts 20-30 40-50
  check from ports 20-10 and then 40-10
        outPorts 20-10 40-10
  check port 25
        outPorts 25
  check from ports 1-65535 and only display failure
        outPorts allf
  check from ports 20-25 and only display success
        outPorts 20-25s
  check from ports 20-25 and display only success, 
  then ports 30-35 and only display failure, 
  then ports 40-50 and display both. 
        outPorts 20-25s 30-35f 40-50
`)
	}
	var w int
	color := flag.Bool("c", false, "add color/bold for success/failure")
	flag.IntVar(&w, "w", 1024, "number of workers to use (use less if you're getting false positives)")
	flag.Parse()
	if *color == true {
		successMsg = GREEN + BOLD + "%s" + NORMAL
		failureMsg = RED + BOLD + "%s" + NORMAL
	} else {
		successMsg = "%s"
		failureMsg = "%s"
	}
	go printLoop()
	for _, arg := range flag.Args() {
		var min, max uint16
		switch arg[len(arg)-1] {
		case 's':
			printFailure = false
			printSuccess = true
			arg = arg[0 : len(arg)-1]
		case 'f':
			printSuccess = false
			printFailure = true
			arg = arg[0 : len(arg)-1]
		}
		if arg == "all" {
			min = 1
			max = 65355
		} else if i := strings.Index(arg, "-"); i != -1 {
			if tmp, err := strconv.ParseUint(arg[:i], 10, 16); err != nil {
				log.Fatal(err)
			} else {
				min = uint16(tmp)
			}
			if tmp, err := strconv.ParseUint(arg[i+1:], 10, 16); err != nil {
				log.Fatal(err)
			} else {
				max = uint16(tmp)
				if min > max {
					tmp := max
					max = min
					min = tmp
				}
			}
		} else {
			if tmp, err := strconv.ParseUint(arg, 10, 16); err != nil {
				log.Fatal(err)
			} else {
				min = uint16(tmp)
				max = min
			}
		}
		wg.Add(w)
		in := make([]chan uint16, w)
		for i := 0; i < w; i++ {
			in[i] = make(chan uint16, w)
			go worker(in[i])
		}
		for i := 0; min <= max; min++ {
			in[i] <- min
			if min == 65535 {
				break
			}
			i++
			if i == w {
				i = 0
			}
		}
		for i := 0; i < w; i++ {
			close(in[i])
		}
		wg.Wait()
		printSuccess = true
		printFailure = true
	}
	close(out)
	<-exit
}

// keeps output from writing over each other; actually happens when its outputting so fast
func printLoop() {
	for m := range out {
		fmt.Println(m)
	}
	exit <- struct{}{}
}

func worker(in <-chan uint16) {
	defer wg.Done()
	for port := range in {
		addr := fmt.Sprintf("portquiz.net:%d", port)
		c, err := d.Dial("tcp", addr)
		if err != nil {
			if printFailure {
				out <- fmt.Sprintf(failureMsg+" on port %d", "failure", port)
			}
		} else {
			if printSuccess {
				c.Close()
				out <- fmt.Sprintf(successMsg+" on port %d", "success", port)
			}
		}
	}
}
