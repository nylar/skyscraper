# Skyscraper

[![GoDoc](https://godoc.org/github.com/nylar/skyscraper?status.svg)](http://godoc.org/github.com/nylar/skyscraper)
[![License](https://img.shields.io/badge/license-CC0-blue.svg)](LICENSE)

A no-frills website scraper

## Example

``` go
package main

import (
    "bufio"
    "bytes"
    "io/ioutil"
    "log"
    "strings"
    "time"

    "github.com/PuerkitoBio/goquery"
    "github.com/nylar/skyscraper"
)

const workers = 10

func main() {
    var domains []string

    data, err := ioutil.ReadFile("sites.txt")
    if err != nil {
        log.Fatalln(err.Error())
    }

    scanner := bufio.NewScanner(bytes.NewBuffer(data))

    for scanner.Scan() {
        line := strings.ToLower(strings.TrimSpace(scanner.Text()))

        if len(line) == 0 || line[0] == '#' {
            continue
        }

        domains = append(domains, line)
    }

    scraper := skyscraper.New(workers)

    for i := 0; i < workers; i++ {
        go scraper.Process()
    }

    go func() {
        for {
            select {
            case response := <-scraper.Out:
                if response.Err != nil {
                    log.Printf("Error scraping %s, error: %s", response.Domain, response.Err.Error())
                } else {
                    doc, err := goquery.NewDocumentFromReader(response.Body)
                    if err != nil {
                        log.Printf("Error parsing %s, error: %s", response.Domain, err.Error())
                    }
                    log.Printf("Title of %s is %q", response.Domain, strings.TrimSpace(doc.Find("title").Text()))
                }
            }
        }
    }()

    go scraper.Add(domains...)

    time.Sleep(time.Second * 10)
    scraper.Close()
    time.Sleep(time.Second * 5)
}
```
