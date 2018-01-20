// Copyright 2017 hIMEI

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"sync"

	"golang.org/x/net/html"
)

var (
	// "Search" subcommand
	searchCmd   = flag.NewFlagSet("search", flag.ExitOnError)
	requestFlag = searchCmd.String("r", "", "your search request to Ichidan")
	// Save output to file
	outputFlag = searchCmd.String("f", "", "save results to file")
	PARSED     []*Host
	FILEPATH   string

	// Version flag gets current app's version
	version    = "0.1.0"
	versionCmd = flag.Bool("v", false, "\tprint current version")

	// usage prints short help message
	usage = func() {
		fmt.Println(BOLD, RED, "\t", "Usage:", RESET)
		fmt.Println(WHT, "\t", "gichidan <command> [<args>] [options]")
		fmt.Println(BLU, "Commands:", GRN, "\t", "search")
		fmt.Println(BLU, "Args:", GRN, "\t", "-r", "\t", CYN, "your search request to Ichidan")
		fmt.Println(BLU, "Options:\n", GRN, "\t\t")
	}

	// helpCmd prints usage()
	helpCmd = flag.Bool("h", false, "\thelp message")
)

// ToFile saves results to given file
func toFile(filepath string, parsed []*Host) {
	dir := path.Dir(filepath)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		errString := BOLD + RED + "Given path does not exist" + RESET
		newerr := errors.New(errString)
		ErrFatal(newerr)
	}

	if _, err := os.Stat(filepath); os.IsExist(err) {
		errString := BOLD + RED + "File already exist, we'll not rewrite it " + RESET
		newerr := errors.New(errString)
		ErrFatal(newerr)
	}

	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE, 0666)
	ErrFatal(err)
	defer file.Close()

	for i, s := range parsed {
		file.WriteString(s.String() + "\n\n\n")
		fmt.Println(i)
	}
}

func main() {
	// Cli options parsing
	flag.Parse()

	if *versionCmd {
		fmt.Println(version)
		os.Exit(1)
	}

	if *helpCmd {
		usage()
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "search":
		searchCmd.Parse(os.Args[2:])
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	if searchCmd.Parsed() {
		if *requestFlag == "" {
			usage()
			os.Exit(1)
		}

		if *outputFlag != "" {
			FILEPATH = *outputFlag
		}
	}

	request := requestProvider(*requestFlag)

	var parsedHosts []*Host
	var recievedNodes []*html.Node
	var mu = &sync.Mutex{}

	// Channels
	channelBody := make(chan *html.Node, BUFFSIZE)
	chanUrls := make(chan string, BUFFSIZE)
	chanHost := make(chan []*Host, BUFFSIZE)

	// Actors
	s := NewSpider()
	p := NewParser()

	totalHosts := 1

	go s.Crawl(request, channelBody)

	for len(parsedHosts) < totalHosts {
		select {
		case recievedNode := <-channelBody:
			if s.checkRoot(recievedNode) == true {
				// Get results total number
				total := s.getTotal(recievedNode)
				totalHosts = toInt(total)
				fmt.Println(BOLD, YEL, "Hosts found: ", CYN, total, RESET)
			}

			go s.getPagination(recievedNode, chanUrls)
			go p.parseOne(recievedNode, chanHost)
			recievedNodes = append(recievedNodes, recievedNode)

		case newUrl := <-chanUrls:
			mu.Lock()
			if s.HandledUrls[newUrl] == false {
				go s.Crawl(newUrl, channelBody)
				s.HandledUrls[newUrl] = true
				SLEEPER()
				fmt.Println(newUrl, " in processing")
			} else {
				//fmt.Println(BOLD, RED, newUrl, "Already visited", RESET)
			}
			mu.Unlock()

		case newhosts := <-chanHost:
			for _, h := range newhosts {
				parsedHosts = append(parsedHosts, h)
				fmt.Println("parsed ", h.HostUrl)
			}
		}
	}

	fmt.Println(BOLD, RED, "Full info:\n", RESET)
	for _, m := range parsedHosts {
		fmt.Println(m.String())
	}

	if FILEPATH != "" {
		fmt.Println(FILEPATH)
		PARSED = parsedHosts
		toFile(FILEPATH, PARSED)
	}

}
