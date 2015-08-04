package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/moovweb/gokogiri"
)

func lcs(s1, s2 string) string {
	var m = make([][]int, 1+len(s1))
	for i := 0; i < len(m); i++ {
		m[i] = make([]int, 1+len(s2))
	}
	longest := 0
	xLongest := 0
	for x := 1; x < 1+len(s1); x++ {
		for y := 1; y < 1+len(s2); y++ {
			if s1[x-1] == s2[y-1] {
				m[x][y] = m[x-1][y-1] + 1
				if m[x][y] > longest {
					longest = m[x][y]
					xLongest = x
				}
			} else {
				m[x][y] = 0
			}
		}
	}
	return s1[xLongest-longest : xLongest]
}

func search(c *http.Client, name string) (foundName, href string, err error) {
	req, err := http.NewRequest("GET", "https://play.google.com/store/search", nil)
	if err != nil {
		err = fmt.Errorf("failed creating request, %s", err)
		return
	}
	values := req.URL.Query()
	values.Add("q", name)
	values.Add("c", "apps")
	req.URL.RawQuery = values.Encode()
	resp, err := c.Do(req)
	if err != nil {
		err = fmt.Errorf("failed searching, %s", err)
		return
	}
	defer resp.Body.Close()
	page, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response body, %s", err)
		return
	}
	doc, err := gokogiri.ParseHtml(page)
	if err != nil {
		err = fmt.Errorf("failed to parse search HTML, %s", err)
		return
	}
	defer doc.Free()
	nodes, err := doc.Search("//div[@id='body-content']//a[@title]")
	if err != nil {
		err = fmt.Errorf("failed to search for app links, %s", err)
		return
	}
	if len(nodes) == 0 {
		err = fmt.Errorf("could not find any app links")
		return
	}
	bestMatchScore := 0
	lowerName := strings.ToLower(name)
	for _, n := range nodes {
		nName := strings.TrimSpace(n.Content())
		score := len(lcs(strings.ToLower(nName), lowerName))
		if score > bestMatchScore {
			foundName = nName
			href = n.Attr("href")
			bestMatchScore = score
		}
	}
	return
}

var numRegexp = regexp.MustCompile(`\D`)

func fetchRating(c *http.Client, path string) (rating float64, ratings int, err error) {
	resp, err := c.Get("https://play.google.com" + path)
	if err != nil {
		err = fmt.Errorf("failed requesting, %s", err)
		return
	}
	defer resp.Body.Close()
	page, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response body, %s", err)
		return
	}
	doc, err := gokogiri.ParseHtml(page)
	if err != nil {
		err = fmt.Errorf("failed to parse search HTML, %s", err)
		return
	}
	defer doc.Free()
	nodes, err := doc.Search("//div[@class='score-container']//div[@class='score']")
	if err != nil {
		err = fmt.Errorf("failed to find score, %s", err)
		return
	}
	if len(nodes) == 0 {
		err = fmt.Errorf("could not find any score")
		return
	}
	if rating, err = strconv.ParseFloat(nodes[0].Content(), 64); err != nil {
		err = fmt.Errorf("could not parse score, %s", err)
		return
	}
	nodes, err = doc.Search("//div[@class='score-container']//span[@class='reviews-num']")
	if err != nil {
		err = fmt.Errorf("failed to find review count, %s", err)
		return
	}
	if len(nodes) == 0 {
		err = fmt.Errorf("could not find review count")
		return
	}
	if ratings, err = strconv.Atoi(
		numRegexp.ReplaceAllString(nodes[0].Content(), ""),
	); err != nil {
		err = fmt.Errorf("could not parse review count, %s", err)
		return
	}
	return
}

func writeCsvRow(out *csv.Writer, name, matched, path string, rating float64, ratings int) {
	url := ""
	if path != "" {
		url = "https://play.google.com" + path
	}
	out.Write([]string{
		name,
		matched,
		url,
		fmt.Sprintf("%f", rating),
		strconv.Itoa(ratings),
	})
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	limit := time.Tick(time.Second)
	client := &http.Client{}
	out := csv.NewWriter(os.Stdout)
	out.Write([]string{
		"Name",
		"Matched",
		"URL",
		"Rating",
		"Ratings",
	})
	for scanner.Scan() {
		<-limit
		name := scanner.Text()
		log.Printf("searching for %s", name)
		foundName, href, err := search(client, name)
		if err != nil {
			log.Printf("failed to find %s, %s", name, err)
			writeCsvRow(out, name, "", "", 0, 0)
			continue
		}
		log.Printf("%s (%s)", foundName, href)

		log.Printf("fetching rating for %s", name)
		<-limit
		rating, ratings, err := fetchRating(client, href)
		if err != nil {
			log.Printf("failed to fetch the rating for %s, %s", name, err)
			writeCsvRow(out, name, foundName, href, 0, 0)
			continue
		}
		log.Printf("%f (%d)", rating, ratings)
		writeCsvRow(out, name, foundName, href, rating, ratings)
	}
	out.Flush()
}
