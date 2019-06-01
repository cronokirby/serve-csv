package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
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

type RawSchema struct {
	Fields []string
	Types  []string
}

// readSchema attempts to read a JSON file's CSV schema.
// This can fail because of IO, or because of an invalid schema.
func readSchema(path string) (*RawSchema, error) {
	var schema RawSchema
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytes, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

type SchemaType int

const (
	INT SchemaType = iota
	STRING
)

type Schema struct {
	Fields []string
	Types  []SchemaType
}

// validate a schema, returning nil if no errors occurred.
// This will check that the schema itself is valid, not whether
// or not it applies to the given CSV file.
func (schema *RawSchema) validate() (*Schema, error) {
	fieldsLen := len(schema.Fields)
	typesLen := len(schema.Types)
	if fieldsLen != typesLen {
		return nil, fmt.Errorf("Mismatched fields and types lengths: %d %d", fieldsLen, typesLen)
	}
	var types []SchemaType
	for _, typeString := range schema.Types {
		var validType SchemaType
		switch typeString {
		case "int":
			validType = INT
		case "string":
			validType = STRING
		default:
			return nil, fmt.Errorf("Unrecognized schema type: %s", typeString)
		}
		types = append(types, validType)
	}
	return &Schema{schema.Fields, types}, nil
}

type CSVData struct {
	rows   [][]interface{}
	Schema *Schema
}

func readCSVData(path string, schema *Schema) (*CSVData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(bufio.NewReader(file))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	rows := make([][]interface{}, 0, len(records))
	for rowI, record := range records {
		recordLen := len(record)
		schemaLen := len(schema.Types)
		if recordLen != schemaLen {
			return nil, fmt.Errorf("row %d: bad record length, expected %d, got %d", rowI, schemaLen, recordLen)
		}
		row := make([]interface{}, len(schema.Types))
		for i, typ := range schema.Types {
			switch typ {
			case INT:
				num, err := strconv.ParseInt(record[i], 10, 64)
				if err != nil {
					return nil, fmt.Errorf("row %d: %v", rowI, err)
				}
				row[i] = num
			case STRING:
				row[i] = record[i]
			}
		}
		rows = append(rows, row)
	}
	data := CSVData{rows, schema}
	return &data, nil
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
		raw, err := readSchema(pair.json)
		if err != nil {
			log.Fatal(err)
		}
		schema, err := raw.validate()
		if err != nil {
			log.Fatal(err)
		}
		data, err := readCSVData(pair.csv, schema)
		fmt.Println(data.rows)
	}
}
