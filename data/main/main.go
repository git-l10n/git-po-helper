package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"
)

func generateLangCode() error {
	tmpMap := make(map[string]string)
	csvFile := "iso-639.csv"
	outFile := "iso-639.go"
	tplFile := "iso-639.t"
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		dirName := filepath.Dir(filepath.Dir(currentFile))
		csvFile = filepath.Join(dirName, csvFile)
		outFile = filepath.Join(dirName, outFile)
		tplFile = filepath.Join(dirName, tplFile)
	}

	f, err := os.Open(csvFile)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if record[0] != "" {
			tmpMap[record[0]] = record[3]
		}
		if record[1] != "" {
			tmpMap[record[1]] = record[3]
		}
		if record[2] != "" {
			tmpMap[record[2]] = record[3]
		}
	}

	t, err := template.ParseFiles(tplFile)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	t.Execute(out, tmpMap)
	return nil
}

func generateLocationCode() error {
	tmpMap := make(map[string]string)
	csvFile := "iso-3166.csv"
	outFile := "iso-3166.go"
	tplFile := "iso-3166.t"
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		dirName := filepath.Dir(filepath.Dir(currentFile))
		csvFile = filepath.Join(dirName, csvFile)
		outFile = filepath.Join(dirName, outFile)
		tplFile = filepath.Join(dirName, tplFile)
	}

	f, err := os.Open(csvFile)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if record[1] != "" {
			tmpMap[record[1]] = record[0]
		}
		if record[2] != "" {
			tmpMap[record[2]] = record[0]
		}
	}

	t, err := template.ParseFiles(tplFile)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	t.Execute(out, tmpMap)
	return nil
}

func main() {
	err := generateLangCode()
	if err == nil {
		err = generateLocationCode()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(-1)
	}
}
