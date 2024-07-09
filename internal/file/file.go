package file

import (
	"log"
	"os"
)

func OpenFile(fileName string) (*os.File, error) {
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
		return nil, err
	}

	// Return the file without closing it
	return file, nil
}
