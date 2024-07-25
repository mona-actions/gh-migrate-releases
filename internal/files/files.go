package files

import (
	"bufio"
	"encoding/json"
	"net/url"
	"os"
	"strings"
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

// read repository list from file assuming each line is a repository
func ReadRepositoryListFromFile(fileName string) ([]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var repositories []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		repo := scanner.Text()
		parsedURL, err := url.Parse(repo)
		if err != nil {
			return nil, err
		}
		path := strings.TrimPrefix(parsedURL.Path, "/")
		repositories = append(repositories, path)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return repositories, nil
}
