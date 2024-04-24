package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ghodss/yaml"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func main() {
	dirname := flag.String("crdDir", "_out/manifests/schema/", "path to directory with crds from where validation field will be parsed")
	outputdir := flag.String("outputDir", "pkg/operator/resources/", "path to dir where go file will be generated")

	flag.Parse()

	files, err := os.ReadDir(*dirname)
	if err != nil {
		panic(fmt.Errorf("error occurred reading directory, %v", err))
	}

	if len(files) == 0 {
		panic("Povided crdDir is empty")
	}

	crds := make(map[string]*extv1.CustomResourceDefinition)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()
		if strings.HasSuffix(filename, ".yaml") {
			crdname, crd := getCRD(filepath.Join(*dirname, filename))
			if crd != nil {
				crds[crdname] = crd
			}
		}
	}
	generateGoFile(*outputdir, crds)
}

var variable = "\t\"%s\": `%s`,\n"

func generateGoFile(outputDir string, crds map[string]*extv1.CustomResourceDefinition) {
	filePath := filepath.Join(outputDir, "crds_generated.go")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	if err != nil {
		panic(fmt.Errorf("failed to create go file %v, %v", filePath, err))
	}
	fmt.Printf("output file: %s\n", file.Name())

	writeOrPanic(file, "package resources\n\n")
	writeOrPanic(file, "// CDICRDs is a map containing yaml strings of all CRDs\n")
	writeOrPanic(file, "var CDICRDs map[string]string = map[string]string{\n")

	crdnames := make([]string, 0)
	for crdname := range crds {
		crdnames = append(crdnames, crdname)
	}
	sort.Strings(crdnames)
	for _, crdname := range crdnames {
		crd := crds[crdname]
		crd.Status = extv1.CustomResourceDefinitionStatus{}
		b, _ := yaml.Marshal(crd)
		writeOrPanic(file, fmt.Sprintf(variable, crdname, strings.ReplaceAll(string(b), "`", "` + \"`\" + `")))
	}
	writeOrPanic(file, "}\n")
}

func getCRD(filename string) (string, *extv1.CustomResourceDefinition) {
	fmt.Printf("reading file: %s\n", filename)
	file, err := os.Open(filename)
	if err != nil {
		panic(fmt.Errorf("failed to read file %v, %v", filename, err))
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	crd := extv1.CustomResourceDefinition{}
	err = k8syaml.NewYAMLToJSONDecoder(file).Decode(&crd)
	if err != nil {
		panic(fmt.Errorf("failed to parse crd from file %v, %v", filename, err))
	}
	return crd.Spec.Names.Singular, &crd
}

func writeOrPanic(f io.StringWriter, str string) {
	if _, err := f.WriteString(str); err != nil {
		panic(fmt.Errorf("failed to write string to file; %w", err))
	}
}
