package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
)

func main() {
	err, flags := parseFlags()
	if err != nil {
		fmt.Println(err.Error())
		flag.Usage()
		os.Exit(1)
	}

	err, testcases := parseTestFiles(flags.TestFiles)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	printTestCases(testcases)
}

type Specification struct {
	Description string
	File        string
}

func printTestCases(testcases []TestCase) error {
	tagMap := make(map[string][]Specification)
	var untagged []Specification

	for _, testcase := range testcases {
		tagRegexes := []string{`(\S+):`}
		r, err := regexp.Compile(fmt.Sprintf("%v", tagRegexes[0]))
		if err != nil {
			return err
		}

		log.Println(testcase.Description)

		specRegex := `[^:]\s*([^:]*)$`
		s, err := regexp.Compile(specRegex)
		if err != nil {
			return err
		}
		spec := Specification{Description: s.FindStringSubmatch(string(testcase.Description))[1], File: testcase.File}

		constainsTag := false
		for _, t := range r.FindAllStringSubmatch(string(testcase.Description), -1) {
			constainsTag = true
			tag := t[1]

			tagMap[tag] = append(tagMap[tag], spec)
		}

		if !constainsTag {
			untagged = append(untagged, spec)
		}
	}

	for tag, specs := range tagMap {
		printSpecs(tag, specs)
	}

	printSpecs("(untagged)", untagged)

	return nil
}

func printSpecs(title string, specs []Specification) {
	if len(specs) == 0 {
		return
	}

	fmt.Println(fmt.Sprintf("%v:", title))
	for _, spec := range specs {
		fmt.Println(fmt.Sprintf("  - %v (%v)", spec.Description, spec.File))
	}
	fmt.Println("")
}

type TestCase struct {
	Description string
	File        string
}

func parseTestFiles(testfiles []string) (error, []TestCase) {
	var testcases []TestCase

	patrolTestRegex := `patrolTest\(\s*['"](.+)['"]`
	r, err := regexp.Compile(fmt.Sprintf("%v", patrolTestRegex))
	if err != nil {
		return err, nil
	}

	for _, testfile := range testfiles {
		b, err := ioutil.ReadFile(testfile)
		if err != nil {
			return err, nil
		}

		for _, testcase := range r.FindAllStringSubmatch(string(b), -1) {
			testcases = append(testcases,
				TestCase{
					Description: testcase[1],
					File:        testfile,
				},
			)
		}
	}

	return nil, testcases
}

type Flags struct {
	TestFiles []string
	Output    *string
}

func parseFlags() (error, Flags) {
	var flags Flags
	//flags.example = flag.String("example", "", "description")

	flag.Parse()
	flags.TestFiles = flag.Args()

	err, valid := validateFlags(flags)
	if !valid {
		return errors.New(fmt.Sprintf("%v", err.Error())), flags
	}
	if err != nil {
		return err, flags
	}

	return nil, flags
}

func validateFlags(flags Flags) (error, bool) {
	if len(flags.TestFiles) == 0 {
		return errors.New("Please specify test files"), false
	}

	return nil, true
}
