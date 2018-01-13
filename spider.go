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
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	//"time"

	"github.com/antchfx/htmlquery"
	"github.com/hIMEI29A/gotorsocks"
	"golang.org/x/net/html"
)

// Consts for connecting to search engine
const (
	ICHIDAN string = "ichidanv34wrx7m7.onion:80"
	SEARCH         = "/search?query="
)

// Spider is an async urls handler
type Spider struct {
	// Urls to crawl
	Url string
	// Urls already being handled
	HandledUrls []string
}

func NewSpider(url string) *Spider {
	var handled []string
	spider := &Spider{url, handled}

	return spider
}

func requestProvider(request string) string {
	var fullRequest string

	switch {
	case string(request[0]) == "/":
		fullRequest = "GET " + request + "\n"

	case string(request[0]) != "/" &&
		string(request[0]) != NONE &&
		strings.Contains(request, NONE) == true:

		splitted := strings.Split(request, NONE)
		fullRequest = "GET " + SEARCH + splitted[0] + "\n"

	default:
		fullRequest = "GET " + SEARCH + request + "\n"
	}

	return fullRequest
}

func connectProvider(url string) net.Conn {
	tor, err := gotorsocks.NewTorGate()
	ErrFatal(err)

	connect, err := tor.DialTor(url)
	ErrFatal(err)

	return connect
}

// getContents makes request to search engine and gets response body
func getContents(request string) *html.Node {
	connect := connectProvider(ICHIDAN)
	defer connect.Close()

	fmt.Fprintf(connect, request)
	resp := bufio.NewReader(connect)

	node, err := htmlquery.Parse(resp)
	ErrFatal(err)

	return node
}

// CheckResult controls empty search results
func checkResult(node *html.Node) bool {
	ch := true

	result := findEntry(node, NORESULT)
	if result != nil {
		ch = false
	}

	return ch
}

func (s *Spider) checkVisited(url string) bool {
	check := false

	for _, handled := range s.HandledUrls {
		if handled == url {
			check = true
			break
		}
	}

	return check
}

func (s *Spider) Crawl(url string, channelBody chan *html.Node, wg *sync.WaitGroup) {

	body := getContents(url)

	if checkResult(body) == false {
		err := errors.New("Nothing found there, Neo!")
		ErrFatal(err)
	}

	channelBody <- body
	s.HandledUrls = append(s.HandledUrls, url)
	fmt.Println("Handled ", url)

	newUrls := s.getPagination(body)

	for _, newurl := range newUrls {
		if s.checkVisited(newurl) == false {
			wg.Add(1)
			go func(url string, channelBody chan *html.Node, wg *sync.WaitGroup) {
				defer wg.Done()
				s.Crawl(newurl, channelBody, wg)
			}(newurl, channelBody, wg)
			SLEEPER()
		}
	}

	return
}

func (s *Spider) checkDone(node *html.Node) bool {
	check := false

	pagination := findEntry(node, PAGINATION)

	if findEntry(pagination, DISABLED) != nil {
		check = true
	}

	return check
}

func (s *Spider) getPagination(node *html.Node) []string {
	var links []string

	pagination := findEntry(node, PAGINATION)

	if pagination != nil {
		current := toInt(getTag(pagination, CURRENT))
		fmt.Println(current)
		hrefs := findEntrys(pagination, LINK)

		for _, newtag := range hrefs {
			if htmlquery.InnerText(newtag) != PREVIOUS &&
				htmlquery.InnerText(newtag) != NEXT &&
				toInt(htmlquery.InnerText(newtag)) > current {
				link := requestProvider(getHref(newtag))
				//				if s.checkVisited(link) == false {
				links = append(links, link)
				//				}
			}
		}
	} else {
		fmt.Println("No more pages")
	}

	return links
}

/*func (s *Spider) Crawl(channelUrl chan string) chan *html.Node {
	channelBody := make(chan *html.Node)
	url := <-chanUrl

	body := getContents(url)

	if checkResult(body) == false {
		err := errors.New("Nothing found there, Neo!")
		ErrFatal(err)
	}

	channelBody <- body

	return channelBody
}

func (s *Spider) getPagination(channelNode chan *html.Node) chan []string {
	var links []string
	channelUrls := make(chan []string)

	node := <-channelNode

	pagination := findEntry(node, PAGINATION)

	if pagination != nil {
		hrefs := findEntrys(pagination, LINK)

		for _, newtag := range hrefs {
			if htmlquery.InnerText(newtag) != "Previous" &&
				htmlquery.InnerText(newtag) != "Next" &&
				htmlquery.InnerText(newtag) != "1" {
				link := requestProvider(getHref(newtag))

				//				if s.checkVisited(link) == false {
				links = append(links, link)
				//				}
			}
		}
	} else {
		fmt.Println("No more pages")
	}

	channelUrls <- links

	return channelUrls
}*/
