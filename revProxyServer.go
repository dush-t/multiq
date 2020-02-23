package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
)

// ReadURLConf reads the file specified by the path
// line by line and returns an array to pointers to
// regexps of the text in each line.
func ReadURLConf(path string) []*regexp.Regexp {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var regexList []*regexp.Regexp
	for scanner.Scan() {
		reg, err := regexp.Compile(scanner.Text())
		if err != nil {
			log.Fatal("Invalid regular expression", scanner.Text())
		}
		regexList = append(regexList, reg)
	}

	return regexList
}
