package roll

import (
	"io"
	"os"
	"path"

	"github.com/konkers/roll/data"
)

func (b *Bot) openFile(filename string) (io.ReadCloser, error) {
	// First try to see if there's a non-archived version.
	file, err := os.Open(filename)
	if err == nil {
		// Found it.
		return file, nil
	}

	// For dev purposes, check the source location
	file, err = os.Open(path.Join("src/github.com/konkers/roll/data", filename))
	if err == nil {
		// Found it.
		return file, nil
	}
	// If that didn't work, try in the archive.
	return data.Assets.Open(filename)
}
