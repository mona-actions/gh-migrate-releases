package files

import (
	"encoding/json"
	"os"
)

func OpenFile(fileName string) (*os.File, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	// Return the file without closing it
	return file, nil
}

func RemoveFile(fileName string) error {
	err := os.Remove(fileName)
	if err != nil {
		return err
	}

	return nil
}

func CreateJSON(data interface{}, filename string) error {
	// Create a new file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a new JSON encoder and write to the file
	encoder := json.NewEncoder(file)
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	return nil
}
