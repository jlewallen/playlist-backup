package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zmb3/spotify"
)

func copyPlaylist(spotifyClient *spotify.Client, userId, sourceName, destinationName string) error {
	destination, err := GetPlaylistByTitle(spotifyClient, userId, destinationName)
	if err != nil {
		return fmt.Errorf("Error getting %s: %v", destinationName, err)
	}
	if destination == nil {
		created, err := spotifyClient.CreatePlaylistForUser(userId, destinationName, "", true)
		if err != nil {
			return fmt.Errorf("Unable to create playlist: %v", err)
		}

		log.Printf("Created destination: %v", created)

		destination, err = GetPlaylistByTitle(spotifyClient, userId, destinationName)
		if err != nil {
			return fmt.Errorf("Error getting %s: %v", destinationName, err)
		}
	} else {
		log.Printf("Found destination: %v", destination)
	}

	main, err := GetPlaylistByTitle(spotifyClient, userId, sourceName)
	if err != nil {
		return fmt.Errorf("Error getting %s: %v", sourceName, err)
	}

	destinationTracks, err := GetPlaylistTracks(spotifyClient, destination.ID)
	if err != nil {
		return fmt.Errorf("Error getting %s: %v", sourceName, err)
	}

	destinationMap := make(map[string]bool)
	for _, track := range destinationTracks {
		destinationMap[track.Track.ID.String()] = true
	}

	mainTracks, err := GetPlaylistTracks(spotifyClient, main.ID)
	if err != nil {
		return fmt.Errorf("Error getting %s: %v", sourceName, err)
	}

	additions := make([]spotify.ID, 0)

	for _, track := range mainTracks {
		if _, ok := destinationMap[track.Track.ID.String()]; !ok {
			additions = append(additions, track.Track.ID)
		}
	}

	if len(additions) > 0 {
		log.Printf("Adding %v tracks", len(additions))

		for i := 0; i < len(additions); i += 50 {
			batch := additions[i:min(i+50, len(additions))]
			_, err := spotifyClient.AddTracksToPlaylist(destination.ID, batch...)
			if err != nil {
				return fmt.Errorf("Error adding tracks: %v", err)
			}
		}
	}

	return nil
}

type options struct {
	Title    string
	Interval int
}

func main() {
	o := options{}

	flag.StringVar(&o.Title, "title", "", "title of the playlist to destination")
	flag.IntVar(&o.Interval, "interval", 60*5, "seconds between destinations")

	flag.Parse()

	if o.Title == "" {
		flag.Usage()
		os.Exit(2)
	}

	for {
		spotifyClient, _ := AuthenticateSpotify()

		user, err := spotifyClient.CurrentUser()
		if err != nil {
			log.Printf("Error: %v", err)
		} else {
			err := copyPlaylist(spotifyClient, user.ID, o.Title, o.Title+" (backup)")
			if err != nil {
				log.Printf("Error: %v", err)
			}

			err = copyPlaylist(spotifyClient, user.ID, o.Title+" (backup)", o.Title)
			if err != nil {
				log.Printf("Error: %v", err)
			}
		}

		log.Printf("Sleep")

		time.Sleep(time.Duration(o.Interval) * time.Second)
	}
}
