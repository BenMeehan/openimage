package provider

import (
	"fmt"
	"io"
	"net/http"
)

func downloadImage(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func parseSize(size string) (int, int) {
	var w, h int
	fmt.Sscanf(size, "%dx%d", &w, &h)
	return w, h
}
