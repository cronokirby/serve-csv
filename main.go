package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// dirFileNames returns a list of file names in a given directory
// errors can happen due to IO, or if the given directory doesn't exist
func dirFileNames(dir string) ([]string, error) {
	file, err := os.Open(dir)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	names, err := file.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	return names, nil
}

type dataPair struct {
	csv  string
	json string
}

func matchDataPairs(paths []string) ([]dataPair, error) {
	var csvs []string
	var jsons []string
	for _, path := range paths {
		if strings.HasSuffix(path, ".csv") {
			csvs = append(csvs, path[:len(path)-len(".csv")])
		}
		if strings.HasSuffix(path, ".json") {
			jsons = append(jsons, path[:len(path)-len(".json")])
		}
	}
	var results []dataPair
	for _, csv := range csvs {
		found := false
		for _, json := range jsons {
			if csv == json {
				found = true
				results = append(results, dataPair{csv + ".csv", json + ".json"})
			}
		}
		if !found {
			return nil, fmt.Errorf("CSV file %s has no corresponding %s.json schema", csv, csv)
		}
	}
	return results, nil
}

func main() {
	args := os.Args[1:]
	dir := args[0]
	fileNames, err := dirFileNames(dir)
	if err != nil {
		log.Fatal(err)
	}
	dataPairs, err := matchDataPairs(fileNames)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dataPairs)
}
