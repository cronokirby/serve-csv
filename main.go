package main

import (
	"fmt"
	"log"
	"os"
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

func main() {
	args := os.Args[1:]
	dir := args[0]
	fileNames, err := dirFileNames(dir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(fileNames)
}
