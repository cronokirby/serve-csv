package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

// matchDataPairs tries to match up dataPairs given a list of paths
// this will error if a CSV file is missing a corresponding JSON schema.
func matchDataPairs(root string, paths []string) ([]dataPair, error) {
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
				csv := fmt.Sprintf("%s/%s.csv", root, csv)
				json := fmt.Sprintf("%s/%s.json", root, json)
				results = append(results, dataPair{csv, json})
			}
		}
		if !found {
			return nil, fmt.Errorf("CSV file %s has no corresponding %s.json schema", csv, csv)
		}
	}
	return results, nil
}

type Schema struct {
	Fields []string
	Types  []string
}

// readSchema attempts to read a JSON file's CSV schema.
// This can fail because of IO, or because of an invalid schema.
func readSchema(path string) (*Schema, error) {
	var schema Schema
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytes, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

func main() {
	args := os.Args[1:]
	dir := args[0]
	fileNames, err := dirFileNames(dir)
	if err != nil {
		log.Fatal(err)
	}
	dataPairs, err := matchDataPairs(dir, fileNames)
	if err != nil {
		log.Fatal(err)
	}
	for _, pair := range dataPairs {
		schema, err := readSchema(pair.json)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(schema)
	}
}
