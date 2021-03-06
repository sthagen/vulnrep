// SPDX-License-Identifier: BSD-3-Clause
//
// Copyright 2019, TIBCO Software Inc. This file is subject to the license
// terms contained in the license file that is distributed with this file.

// Usage: vulnrepconv -input <fname> [-output <fname>]
//
// The tool vulnrepconv converts vulnerability reports to/from CVRF(xml) and CSAF(json)
// formats. Note that there are some features of the JSON format not available in the XML
// format (notably, translation support), so this tool cannot fully convert a document
// from JSON --> XML --> JSON, but it will get about as close as possible. Round
// trips from XML --> JSON --> XML will work. See the Limitations section for details.
//
// The input file is required. If no output file is indicated, then the output is
// written to STDOUT.
//
// The type of file is assumed based on file extension. Input and output files
// must have either a ".xml" or a ".json" extension.
//
// Limitations
//
// Round trip conversions from XML-->JSON-->XML work, but may not be identical, due
// to a number of factors:
// - namespace prefixes might change
// - whitespaces might change
// - attribute ordering might change
// - xml:lang attributes will be discarded
// - CDATA and comments may change
//
// A carefully constructed XML file, however, will be able to successfully round-trip
// back to a byte-for-byte identical file.
//
// See Also
//
// The tool is primarily a command line wrapper around the vulnrep package. For
// details about vulnerability representation, please see that package.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TIBCOSoftware/vulnrep"
)

type toConvert struct {
	input        string
	output       string
	help         bool
	dontContinue bool
}

func (tc *toConvert) parseArgs(appName string, args []string) error {

	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.StringVar(&tc.input, "input", "", "file to parse for conversion")
	fs.StringVar(&tc.output, "output", "", "output destination")
	fs.BoolVar(&tc.help, "h", false, "set this flag for help.")

	err := fs.Parse(args)
	if err != nil {
		return err
	}
	if tc.help {
		fs.PrintDefaults()
		tc.dontContinue = true
		return nil
	}
	if tc.input == "" {
		return fmt.Errorf("must specify an input file with the -input parameter")
	}

	return nil
}

func parseXMLFile(fName string) (vulnrep.Report, error) {
	// no security risk to parsing XML here.
	f, err := os.Open(fName) //nolint:gosec
	if err != nil {
		return vulnrep.Report{}, err
	}
	defer safeClose(f)

	return vulnrep.ParseXML(f)

}
func parseJSONFile(fName string) (vulnrep.Report, error) {
	// no security risk to parsing JSON here.
	f, err := os.Open(fName) //nolint:gosec
	if err != nil {
		return vulnrep.Report{}, err
	}
	defer safeClose(f)

	return vulnrep.ParseJSON(f)

}

func (tc *toConvert) doConversion() (err error) {

	var readFunc func(string) (vulnrep.Report, error)

	ext := filepath.Ext(tc.input)
	switch ext {
	case ".xml":
		readFunc = parseXMLFile
	case ".json":
		readFunc = parseJSONFile
	default:
		return fmt.Errorf("unrecognized file extension %v - don't know how to parse file", ext)
	}

	rep, err := readFunc(tc.input)
	if err != nil {
		return err
	}

	outFunc := rep.ToCSAF
	var out io.Writer
	if tc.output == "" {
		out = os.Stdout
	} else {
		outFile, err := os.Create(tc.output)
		if err != nil {
			return fmt.Errorf("unable to open output file for conversion: %v", err)
		}
		defer safeWriteClose(&err, outFile)
		out = outFile

		switch filepath.Ext(tc.output) {
		case ".xml":
			outFunc = rep.ToCVRF
		default:
			outFunc = rep.ToCSAF
		}
	}

	return outFunc(out)
}

func (tc *toConvert) run(appName string, args []string) error {
	err := tc.parseArgs(appName, args)
	if err != nil {
		return err
	}

	if tc.dontContinue {
		return nil
	}
	return tc.doConversion()
}

func safeClose(rc io.Closer) {
	rc.Close() //nolint:errcheck,gosec
}

// safeWriteCloser makes sure that any errors that occur upon closing the file
// are returned, if the function would otherwise return nil. This is meant to
// be called as part of a deferred close operation.
//
// Note that the caller will need to use named returns in order to pass the
// correct *error to this method.
func safeWriteClose(err *error, wc io.Closer) {
	newErr := wc.Close()
	if *err == nil {
		*err = newErr
	}
}

func main() {

	app := &toConvert{}
	err := app.run(os.Args[0], os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
