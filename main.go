package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type RepoInfo struct {
	Name      string
	Stars     int
	PackageID string
}

type InputRepo struct {
	RepoName        string
	PackageID       string
	DependentsAfter string
}

func saveToFile(filename string, data string) {
	f, err := os.OpenFile(filename+".txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(data); err != nil {
		log.Fatal(err)
	}
}

func scrapeRepo(inputRepo InputRepo, wg *sync.WaitGroup) {
	defer wg.Done()

	baseURL := "https://github.com/" + inputRepo.RepoName
	if inputRepo.PackageID != "" {
		baseURL += "/network/dependents?package_id=" + inputRepo.PackageID
	} else {
		baseURL += "/network/dependents"
	}

	// Check if DependentsAfter is set and append it to baseURL
	if inputRepo.DependentsAfter != "" {
		separator := "&"
		if !strings.Contains(baseURL, "?") {
			separator = "?"
		}
		baseURL += separator + "dependents_after=" + inputRepo.DependentsAfter
	}
	minStarsCnt := 1000
	var result []RepoInfo
	nextExists := true

	for nextExists {
		fmt.Println("url:", baseURL, " cnt:", len(result))

		resp, err := http.Get(baseURL)
		if err != nil {
			log.Fatal(err)
			return
		}
		defer resp.Body.Close()

		document, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
			return
		}

		document.Find("div.Box-row").Each(func(index int, element *goquery.Selection) {
			repoName := element.Find("a[data-repository-hovercards-enabled]").Text() + "/" + element.Find("a[data-hovercard-type=repository]").Text()
			starsStr := strings.ReplaceAll(strings.TrimSpace(element.Find("svg.octicon-star").Parent().Text()), ",", "")
			stars, err := strconv.Atoi(starsStr)
			if err == nil && stars > minStarsCnt {
				result = append(result, RepoInfo{Name: repoName, Stars: stars})
				data := fmt.Sprintf("%s, %d\n", repoName, stars)
				repoName := strings.ReplaceAll(inputRepo.RepoName, "/", "-")
				saveToFile(repoName, data)
			}
		})

		nextExists = false
		paginateContainer := document.Find("div.paginate-container")
		paginateContainer.Find("a").Each(func(index int, item *goquery.Selection) {
			if item.Text() == "Next" {
				nextExists = true
				baseURL, _ = item.Attr("href")
			}
		})

		if !nextExists {
			fmt.Println("waiting for 10 seconds...")
			time.Sleep(10 * time.Second)
			nextExists = true
		}
	}
}

func main() {
	repos := []InputRepo{
		{RepoName: "vercel/next.js", PackageID: "UGFja2FnZS0xNDIzMDMwOA%3D%3D", DependentsAfter: "MjI3MjUxNDU5OTI"},
		{RepoName: "prisma/prisma", DependentsAfter: "NzkzNjY1NTI2"},
		{RepoName: "aws/aws-sdk-js-v3", DependentsAfter: "MjY3ODg4NjYyMDI"},
	}

	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go scrapeRepo(repo, &wg)
	}

	wg.Wait()
}
