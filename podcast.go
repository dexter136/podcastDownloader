package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type podcastDef struct {
	Url           string
	Lasttime      int64
	Titleoverride string
}

type Episode struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	TimeStamp int64  `json:"timestamp"`
	Filepath  string
}

func readConfig(filename string) ([]podcastDef, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Can't get config %v", err)
	}
	var podcasts []podcastDef
	err = yaml.Unmarshal(file, &podcasts)
	if err != nil {
		return nil, fmt.Errorf("Can't read configfile %v", err)
	}
	return podcasts, nil
}

func getEpisodes(URL string, lastTime int64, titleOverride string) (title string, episodes []*Episode) {
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(URL)
	reg, _ := regexp.Compile(`[\\\/:"*?<>|]`)
	title = feed.Title
	if titleOverride != "" {
		title = titleOverride
	}
	for _, item := range feed.Items {

		if item.PublishedParsed.Unix() < lastTime {
			return
		}
		epName := reg.ReplaceAllString(item.Title, "")
		extension := getFileExtension(item.Enclosures[0].URL, epName)
		epPath := filepath.Join("/podcasts", title, epName+extension)
		episodes = append(episodes, &Episode{
			Name:      item.Title,
			URL:       item.Enclosures[0].URL,
			TimeStamp: item.PublishedParsed.Unix(),
			Filepath:  epPath,
		})
	}
	return
}

func makeDirectory(title string) error {
	err := os.Mkdir(filepath.Join("/podcasts", title), 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func getFileExtension(url, eptitle string) (extension string) {
	extension = filepath.Ext(url)
	if extension == "" {
		log.Print("Assuming .mp3 extension for %s", eptitle)
		return ".mp3"
	}
	return strings.Split(extension, "?")[0]
}

func downloadEpisode(url, filePath, epTitle string) error {
	log.Print("Downloading ", epTitle)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.ReadFrom(resp.Body)
	return nil
}

func checkEpisodesExist(episodesList []*Episode, title string) (episodes []*Episode) {
	for _, e := range episodesList {
		_, err := os.Stat(e.Filepath)
		if !os.IsNotExist(err) {
			continue
		}
		episodes = append(episodes, e)
	}
	return episodes
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func downloadPodcast(episodes []*Episode, title string, maxDownload int) error {
	err := makeDirectory(title)
	if err != nil {
		return err
	}
	initialSize := len(episodes)
	log.Print("Found ", initialSize, " episodes for ", title)
	episodes = checkEpisodesExist(episodes, title)
	log.Print("Skipping ", initialSize-len(episodes), " episodes that already exist")

	for _, e := range episodes[:Min(maxDownload, len(episodes))] {
		err := downloadEpisode(e.URL, e.Filepath, e.Name)
		if err != nil {
			log.Print("Error getting episode ", e.Name)
			log.Print(err)
		}
	}
	return nil
}

func getPodcast(podcast podcastDef, maxDownload int) error {
	title, episodes := getEpisodes(podcast.Url, podcast.Lasttime, podcast.Titleoverride)
	err := downloadPodcast(episodes, title, maxDownload)
	return err
}

func main() {
	podcasts, err := readConfig("config/config.yaml")
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, podcast := range podcasts {
		err := getPodcast(podcast, 10)

		if err != nil {
			fmt.Println(err)
		}
	}
}
