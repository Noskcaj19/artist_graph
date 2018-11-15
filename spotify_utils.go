package main

import "github.com/zmb3/spotify"

// GetAllPlaylistTracks returns all paged playlist tracks
func GetAllPlaylistTracks(playlist spotify.ID) ([]spotify.PlaylistTrack, error) {
	var allTracks []spotify.PlaylistTrack
	var total int
	limit := 100
	offset := 0
	opt := spotify.Options{
		Limit:  &limit,
		Offset: &offset,
	}
	for {
		page, err := sp.GetPlaylistTracksOpt(playlist, &opt, "")
		if err != nil {
			return nil, err
		}
		total = page.Total
		allTracks = append(allTracks, page.Tracks...)
		offset = offset + limit
		if total < offset {
			break
		}
	}

	return allTracks, nil
}
