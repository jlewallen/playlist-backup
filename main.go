package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/zmb3/spotify"
)

func getPlaylistByTitle(spotifyClient *spotify.Client, name string) (*spotify.SimplePlaylist, error) {
	limit := 20
	offset := 0
	options := spotify.Options{Limit: &limit, Offset: &offset}
	for {
		playlists, err := spotifyClient.GetPlaylistsForUserOpt("jlewalle", &options)
		if err != nil {
			return nil, err
		}

		for _, iter := range playlists.Playlists {
			if strings.EqualFold(iter.Name, name) {
				return &iter, nil
			}
		}

		if len(playlists.Playlists) < *options.Limit {
			break
		}

		offset := *options.Limit + *options.Offset
		options.Offset = &offset
	}

	return nil, nil
}

func getPlaylistTracks(spotifyClient *spotify.Client, userId string, id spotify.ID) (allTracks []spotify.PlaylistTrack, err error) {
	limit := 100
	offset := 0
	options := spotify.Options{Limit: &limit, Offset: &offset}
	for {
		tracks, spotifyErr := spotifyClient.GetPlaylistTracksOpt(userId, id, &options, "")
		if spotifyErr != nil {
			err = spotifyErr
			return
		}

		allTracks = append(allTracks, tracks.Tracks...)

		if len(tracks.Tracks) < *options.Limit {
			break
		}

		offset := *options.Limit + *options.Offset
		options.Offset = &offset
	}

	return
}

func backupPlaylist(spotifyClient *spotify.Client, userId, name string) error {
	backupName := name + " (backup)"
	backup, err := getPlaylistByTitle(spotifyClient, backupName)
	if err != nil {
		return fmt.Errorf("Error getting %s: %v", backupName, err)
	}
	if backup == nil {
		created, err := spotifyClient.CreatePlaylistForUser(userId, backupName, true)
		if err != nil {
			return fmt.Errorf("Unable to create playlist: %v", err)
		}

		log.Printf("Created backup: %v", created)

		backup, err = getPlaylistByTitle(spotifyClient, backupName)
		if err != nil {
			return fmt.Errorf("Error getting %s: %v", backupName, err)
		}
	} else {
		log.Printf("Found backup: %v", backup)
	}

	main, err := getPlaylistByTitle(spotifyClient, name)
	if err != nil {
		return fmt.Errorf("Error getting %s: %v", name, err)
	}

	backupTracks, err := getPlaylistTracks(spotifyClient, userId, backup.ID)
	if err != nil {
		return fmt.Errorf("Error getting %s: %v", name, err)
	}

	backupMap := make(map[string]bool)
	for _, track := range backupTracks {
		backupMap[track.Track.ID.String()] = true
	}

	mainTracks, err := getPlaylistTracks(spotifyClient, userId, main.ID)
	if err != nil {
		return fmt.Errorf("Error getting %s: %v", name, err)
	}

	additions := make([]spotify.ID, 0)

	for _, track := range mainTracks {
		if _, ok := backupMap[track.Track.ID.String()]; !ok {
			additions = append(additions, track.Track.ID)
		}
	}

	if len(additions) > 0 {
		log.Printf("Adding %v tracks", len(additions))

		for i := 0; i < len(additions); i += 50 {
			batch := additions[i:min(i+50, len(additions))]
			_, err := spotifyClient.AddTracksToPlaylist(userId, backup.ID, batch...)
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

	flag.StringVar(&o.Title, "title", "", "title of the playlist to backup")
	flag.IntVar(&o.Interval, "interval", 60*5, "seconds between backups")

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
			err := backupPlaylist(spotifyClient, user.ID, o.Title)
			if err != nil {
				log.Printf("Error: %v", err)
			}
		}

		log.Printf("Sleep")

		time.Sleep(time.Duration(o.Interval) * time.Second)
	}
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
