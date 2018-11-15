package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/zmb3/spotify"
)

type songSpec struct {
	songTitle string
	artist    spotify.SimpleArtist
	others    []spotify.SimpleArtist
}

var sp spotify.Client

func findPlaylist(name string) (spotify.SimplePlaylist, error) {
	playlists, err := sp.CurrentUsersPlaylists()
	if err != nil {
		log.Fatal("Error fetching playlist:", err)
	}
	for _, playlist := range playlists.Playlists {
		if playlist.Name == name {
			return playlist, nil
		}
	}
	return spotify.SimplePlaylist{}, errors.New("No matching playlist")
}

func collectArtists(tracks *[]spotify.PlaylistTrack) []songSpec {
	var artists []songSpec
	for _, track := range *tracks {
		artists = append(artists, songSpec{track.Track.Name, track.Track.Artists[0], track.Track.Artists[1:]})
	}
	return artists
}

func processPlaylist(playlist spotify.SimplePlaylist) {
	fmt.Printf("Using playlist: %s, (%d tracks)\n", playlist.Name, playlist.Tracks.Total)
	tracks, err := GetAllPlaylistTracks(playlist.ID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error fetching tracks:", err)
	}
	fmt.Println(len(tracks))
	artists := collectArtists(&tracks)
	fmt.Println("digraph {")
	for _, artist := range artists {
		for _, other := range artist.others {
			fmt.Printf("  \"%s\" -> \"%s\" [label=\"%s\"];\n", artist.artist.Name, other.Name, strings.Replace(artist.songTitle, "\"", "", -1))
		}
	}
	fmt.Println("}")
}

func main() {
	sp = getClient()
	user, err := sp.CurrentUser()
	if err != nil {
		fmt.Println("Auth not valid?", err)
		os.Exit(1)
	}

	fmt.Println("User:", user.DisplayName)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: provide a playlist name:")
		playlists, err := sp.CurrentUsersPlaylists()
		if err != nil {
			log.Fatal("Unable to get user playlists:", err)
		}
		for _, playlist := range playlists.Playlists {
			fmt.Println(playlist.Name)
		}
		return
	}

	playlist, err := findPlaylist(os.Args[1])
	if err != nil {
		log.Fatal("No matching playlist")
	}
	processPlaylist(playlist)
}
