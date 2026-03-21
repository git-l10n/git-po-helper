package main

import (
	"encoding/csv"
	"fmt"
	htmltemplate "html/template"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	texttemplate "text/template"

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

	t, err := htmltemplate.ParseFiles(tplFile)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	return t.Execute(out, tmpMap)
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

	t, err := htmltemplate.ParseFiles(tplFile)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	return t.Execute(out, tmpMap)
}

func generateScriptCode() error {
	tmpMap := make(map[string]string)
	csvFile := "iso-15924.csv"
	outFile := "iso-15924.go"
	tplFile := "iso-15924.t"
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
	records, err := r.ReadAll()
	if err != nil {
		return err
	}
	// Skip header
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 3 {
			continue
		}
		code, numeric, name := record[0], record[1], record[2]
		if code != "" {
			tmpMap[code] = name
		}
		if numeric != "" {
			tmpMap[numeric] = name
		}
	}

	t, err := texttemplate.New(filepath.Base(tplFile)).Funcs(texttemplate.FuncMap{
		"goescape": func(s string) string {
			return strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`)
		},
	}).ParseFiles(tplFile)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	return t.Execute(out, tmpMap)
}

func main() {
	err := generateLangCode()
	if err == nil {
		err = generateLocationCode()
	}
	if err == nil {
		err = generateScriptCode()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(-1)
	}
}
