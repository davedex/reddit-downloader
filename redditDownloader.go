// A simple script to search a subreddit and download youtube links over a certain duration
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/jzelinskie/geddit"
	"github.com/mvdan/xurls"
)

type Configuration struct {
	User        string
	Password    string
	Entries     int
	MinDuration int
}

func main() {
	// Load the config
	configFile := path.Join(os.Getenv("HOME"), ".config", "redditDownloader.conf")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("Create a config file at ~/.config/redditDownloader.conf. A json message with user and password strings as well as a an entry count of reddit to search and the minimum duration of videos to download.")
		os.Exit(1)
	}
	file, err := os.Open(configFile)
	if err != nil {
		fmt.Println("Unable to open the config file")
		os.Exit(1)
	}
	decoder := json.NewDecoder(file)
	config := Configuration{}
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("Unable to read config file:", err)
		os.Exit(1)
	}

	// Connect to reddit
	session, err := geddit.NewLoginSession(
		config.User,
		config.Password,
		"gedditAgent v1",
	)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Set listing options
	subOpts := geddit.ListingOptions{
		Limit: config.Entries,
	}

	// Get reddit's default frontpage
	// submissions, _ := session.DefaultFrontpage(geddit.DefaultPopularity, subOpts)

	// Get our own personal frontpage
	// submissions, _ = session.Frontpage(geddit.DefaultPopularity, subOpts)

	// Get specific subreddit submissions, sorted by new
	submissions, err := session.SubredditSubmissions("rugbyunion", geddit.NewSubmissions, subOpts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	for _, s := range submissions {
		// fmt.Printf("Title: %s\nAuthor: %s\nUrl: %s\n", s.Title, s.Author, s.URL)
		fmt.Println(s.Title)
		if strings.Contains(s.URL, "youtube") {
			wg.Add(1)
			download(s.URL, wg, config.MinDuration)
		}
		// Check the comments as well
		comments, err := session.Comments(s)
		if err != nil {
			fmt.Println(err)
		}
		for _, c := range comments {
			urls := xurls.Strict.FindAllString(c.Body, -1)
			for _, u := range urls {
				if strings.Contains(u, "youtube") {
					wg.Add(1)
					download(u, wg, config.MinDuration)
				}
			}
		}
	}
}

func download(u string, wg sync.WaitGroup, minDuration int) {
	duration := exe_cmd("/usr/local/bin/youtube-dl --get-duration "+u, &wg)
	fmt.Println(u, " - ", duration)
	timesplit := strings.Split(duration, ":")
	if len(timesplit) > 2 {
		fmt.Println("\tDownloading as over one hour")
		wg.Add(1)
		exe_cmd("/usr/local/bin/youtube-dl -o /media/seagate_four_zero/videos/TV/Rugby/%(title)s.%(ext)s -w "+u, &wg)
	} else if len(timesplit) == 2 {
		i, err := strconv.Atoi(timesplit[0])
		if err != nil {
			fmt.Println(err)
		}
		if i >= minDuration {
			fmt.Println("\tDownloading as over configured minimum duration: " + strconv.Itoa(minDuration))
			wg.Add(1)
			exe_cmd("/usr/local/bin/youtube-dl -o /media/seagate_four_zero/videos/TV/Rugby/%(title)s.%(ext)s -w "+u, &wg)
		} else {
			fmt.Println("\tNot downloading as not over minimum duration: " + strconv.Itoa(i) + "<" + strconv.Itoa(minDuration))
		}
	}
}

func exe_cmd(cmd string, wg *sync.WaitGroup) string {
	// fmt.Println("command is ", cmd)
	// splitting head => g++ parts => rest of the command
	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	// fmt.Printf("%s", out)
	wg.Done() // Need to signal to waitgroup that this goroutine is done
	return strings.TrimSpace(string(out))
}
