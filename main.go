package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
)

func main() {
	err, flags := parseFlags()
	if err != nil {
		printUsage()
		fmt.Println("")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err, testcases := parseTestFiles(flags.TestFiles)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err, requirements := parseRequirementsFile(*flags.RequirementsFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	requirementLookupMap := createRequirementLookupMap(requirements)

	err, specificationMap, untaggedSpecifications := createSpecificationMap(testcases)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	printSpecificationMap(specificationMap, untaggedSpecifications, requirementLookupMap)

	if *flags.Store {
		println(fmt.Sprintf("Storing specification map in %v", *flags.SpecificationsMapOutput))
		err = storeSpecificationMap(*flags.SpecificationsMapOutput, specificationMap, untaggedSpecifications, requirementLookupMap)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}
}

type Requirement struct {
	Tag         string
	Description string
}

func createRequirementLookupMap(requirements []Requirement) map[string]Requirement {
	requirementLookupMap := make(map[string]Requirement)
	for _, requirement := range requirements {
		requirementLookupMap[requirement.Tag] = requirement

	}
	return requirementLookupMap
}

func parseRequirementsFile(file string) (error, []Requirement) {
	var requirements []Requirement
	f, err := os.Open(file)
	if err != nil {
		return err, nil
	}
	defer f.Close()

	requirementsRegex := `\|\s*(\w*)\s*\|\s*([\s\w]+)\s*\|`
	r, err := regexp.Compile(requirementsRegex)
	if err != nil {
		return err, nil
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		match := r.FindStringSubmatch(scanner.Text())
		if len(match) != 0 {
			if match[1] == "Tag" || match[1] == "tag" {
				continue
			}
			if match[2] == "Description" || match[2] == "description" {
				continue
			}
			requirements = append(requirements, Requirement{Tag: strings.TrimSpace(match[1]), Description: strings.TrimSpace(match[2])})
		}
	}

	return nil, requirements
}

type Specification struct {
	Description string
	File        string
}

func createSpecificationMap(testcases []TestCase) (error, map[string][]Specification, []Specification) {
	specificationMap := make(map[string][]Specification)
	var untagged []Specification

	for _, testcase := range testcases {
		tagRegexes := []string{`(\S+):`}
		r, err := regexp.Compile(fmt.Sprintf("%v", tagRegexes[0]))
		if err != nil {
			return err, specificationMap, untagged
		}

		specRegex := `[^:]\s*([^:]*)$`
		s, err := regexp.Compile(specRegex)
		if err != nil {
			return err, specificationMap, untagged
		}
		spec := Specification{Description: s.FindStringSubmatch(string(testcase.Description))[1], File: testcase.File}

		constainsTag := false
		for _, t := range r.FindAllStringSubmatch(string(testcase.Description), -1) {
			constainsTag = true
			tag := t[1]

			specificationMap[tag] = append(specificationMap[tag], spec)
		}

		if !constainsTag {
			untagged = append(untagged, spec)
		}
	}
	return nil, specificationMap, untagged
}

func storeSpecificationMap(filepath string, specificationMap map[string][]Specification, untagged []Specification, requirementLookupMap map[string]Requirement) error {
	var markdown string
	markdown += "# Specifications Map (generated)\n\n"

	for tag, specs := range specificationMap {
		req := tag
		if val, ok := requirementLookupMap[tag]; ok {
			req = fmt.Sprintf("%v: %v", tag, val.Description)
		}
		markdown += fmt.Sprintf("- **%v**\n", req)
		for _, spec := range specs {
			markdown += fmt.Sprintf("    - %v *(%v)*\n", spec.Description, spec.File)
		}
	}

	if len(untagged) != 0 {
		markdown += fmt.Sprintf("- **%v**\n", "(untagged)")
		for _, spec := range untagged {
			markdown += fmt.Sprintf("    - %v *(%v)*\n", spec.Description, spec.File)
		}
	}

	err := os.MkdirAll(path.Dir(filepath), 0700)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(markdown); err != nil {
		return err
	}

	return nil
}

func printSpecificationMap(specificationMap map[string][]Specification, untagged []Specification, requirementLookupMap map[string]Requirement) error {
	for tag, specs := range specificationMap {
		req := tag
		if val, ok := requirementLookupMap[tag]; ok {
			req = fmt.Sprintf("%v: %v", tag, val.Description)
		}
		printSpecs(req, specs)
	}

	printSpecs("(untagged)", untagged)

	return nil
}

func printSpecs(title string, specs []Specification) {
	if len(specs) == 0 {
		return
	}

	fmt.Println(fmt.Sprintf("%v", title))
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

func printUsage() {
	fmt.Printf("Usage: %s [OPTIONS] testfile1 testfile2 ...\n", path.Base(os.Args[0]))
	flag.PrintDefaults()
}

type Flags struct {
	Store                   *bool
	TestFiles               []string
	RequirementsFile        *string
	SpecificationsMapOutput *string
	RisksFile               *string
	RiskTableOutput         *string
	Output                  *string
}

func parseFlags() (error, Flags) {
	var flags Flags
	flags.RequirementsFile = flag.String("reqs", "", "Path to file with requirements")
	flags.SpecificationsMapOutput = flag.String("spec-map", "docs/specifications-map.md", "Filepath for output of specification map")
	//flags.RisksFile = flag.String("risks", "", "Path to file with risks")
	//flags.RisksFile = flag.String("risks-table", "docs/risks-table.md", "Filepath for output of risks map")
	flags.Store = flag.Bool("store", false, "Wether to store the output to disk")

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
