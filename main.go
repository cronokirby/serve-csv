package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin"
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

type dataPath struct {
	route string
	csv   string
	json  string
}

// matchDataPaths tries to match up dataPairs given a list of paths
// this will error if a CSV file is missing a corresponding JSON schema.
func matchDataPaths(root string, paths []string) ([]dataPath, error) {
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
	var results []dataPath
	for _, csv := range csvs {
		found := false
		for _, json := range jsons {
			if csv == json {
				found = true
				route := csv
				csv := fmt.Sprintf("%s%s.csv", root, csv)
				json := fmt.Sprintf("%s%s.json", root, json)
				results = append(results, dataPath{route, csv, json})
			}
		}
		if !found {
			return nil, fmt.Errorf("CSV file %s has no corresponding %s.json schema", csv, csv)
		}
	}
	return results, nil
}

// RawSchema holds the raw structure of a schema
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

// SchemaType represents the valid types for CSV fields
type SchemaType int

const (
	// INT represents an integer field
	INT SchemaType = iota
	// STRING represents a string field
	STRING
)

// Schema holds a set of Fields and corresponding Types
// Unlike RawSchema, we've made sure these have the same length,
// and that all the declared types are valid.
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

// CSVData holds the data contained in a CSV file.
type CSVData struct {
	rows   [][]interface{}
	Schema *Schema
}

// readCSVData will read the data in a file, checking it against a schema
// This will return an error as soon as any row doesn't match the given schema.
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
			return nil, fmt.Errorf("%s: row %d: bad record length, expected %d, got %d", path, rowI, schemaLen, recordLen)
		}
		row := make([]interface{}, len(schema.Types))
		for i, typ := range schema.Types {
			switch typ {
			case INT:
				num, err := strconv.ParseInt(record[i], 10, 64)
				if err != nil {
					return nil, fmt.Errorf("%s: row %d: %v", path, rowI, err)
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

// jsonNth returns the nth row of data as a JSON byte string.
func (data CSVData) jsonNth(index int) ([]byte, error) {
	if index < 0 || index >= len(data.rows) {
		return nil, fmt.Errorf("index %d out of bounds", index)
	}
	row := data.rows[index]
	mp := make(map[string]interface{})
	for i, val := range row {
		mp[data.Schema.Fields[i]] = val
	}
	json, err := json.Marshal(mp)
	// If we can't encode the json, this is a problem with how our schema is designed
	if err != nil {
		panic(err)
	}
	return json, nil
}

// jsonAll returns a JSON array containing all the data
func (data *CSVData) jsonAll() []byte {
	rows := make([]map[string]interface{}, 0, len(data.rows))
	for _, row := range data.rows {
		mp := make(map[string]interface{})
		for i, val := range row {
			mp[data.Schema.Fields[i]] = val
		}
		rows = append(rows, mp)
	}
	json, err := json.Marshal(rows)
	// We should always be able to encode our json
	if err != nil {
		panic(err)
	}
	return json
}

// DataRoutes matches up route paths to CSVData
type DataRoutes struct {
	routes map[string]CSVData
}

// NewDataRoutes creates a new DataRoutes struct
// This is necessary since the zero value can't be used.
func NewDataRoutes() *DataRoutes {
	return &DataRoutes{routes: make(map[string]CSVData)}
}

// Insert adds a new batch of CSVData
func (routes *DataRoutes) Insert(route string, data CSVData) {
	routes.routes[route] = data
}

// GetAll returns a JSON blob holding all the rows of a route
func (routes *DataRoutes) GetAll(route string) ([]byte, error) {
	data, ok := routes.routes[route]
	if !ok {
		return nil, fmt.Errorf("Unknown route: %s", route)
	}
	return data.jsonAll(), nil
}

// GetNth returns a JSON blob for the nth item of a route
func (routes *DataRoutes) GetNth(route string, index int) ([]byte, error) {
	data, ok := routes.routes[route]
	if !ok {
		return nil, fmt.Errorf("Unknown route: %s", route)
	}
	return data.jsonNth(index)
}

var (
	dir  = kingpin.Arg("dir", "The directory to serve").Required().String()
	port = kingpin.Flag("port", "The port to listen on").Default("1234").Short('p').String()
)

func main() {
	kingpin.Parse()
	fileNames, err := dirFileNames(*dir)
	if err != nil {
		log.Fatal(err)
	}
	dataPaths, err := matchDataPaths(*dir, fileNames)
	if err != nil {
		log.Fatal(err)
	}
	routes := NewDataRoutes()
	for _, path := range dataPaths {
		raw, err := readSchema(path.json)
		if err != nil {
			log.Fatal(err)
		}
		schema, err := raw.validate()
		if err != nil {
			log.Fatal(fmt.Sprintf("Error validating %s: %v", path.json, err))
		}
		data, err := readCSVData(path.csv, schema)
		if err != nil {
			log.Fatal(err)
		}
		routes.Insert(path.route, *data)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// this removes the first /
		route := r.URL.Path[1:]
		splits := strings.Split(route, "/")
		lastPart := splits[len(splits)-1]
		index, indexErr := strconv.ParseInt(lastPart, 10, 32)
		var data []byte
		var err error
		if indexErr != nil {
			data, err = routes.GetAll(route)
		} else {
			route := route[:len(route)-len(lastPart)-1]
			data, err = routes.GetNth(route, int(index))
		}
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "%v"}`, err)
			return
		}
		w.Write(data)
	})
	log.Printf("Listening on port %s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
