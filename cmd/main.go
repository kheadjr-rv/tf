package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	tfjson "github.com/hashicorp/terraform-json"

	address "github.com/hashicorp/go-terraform-address"

	"github.com/kheadjr-rv/tfwriter/tfwriter/schemamd"
)

type schemas struct {
	tfjson.ProviderSchemas
}

func main() {

	f, err := ioutil.ReadFile("../data/schemas.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s := schemas{}

	err = json.Unmarshal(f, &s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// fmt.Println(s.Providers())

	a, err := address.Parse("main.tf", []byte("aws_acm_certificate.arn"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(a.(*address.Address).ResourceSpec.String())

	err = schemamd.Render(s.Schemas["aws"].ConfigSchema, os.Stdout)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (s *schemas) Providers() []string {
	names := make([]string, 0, len(s.Schemas))
	for name := range s.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *schemas) Resources(provider string, ty string) ([]string, error) {
	if _, ok := s.Schemas[provider]; !ok {
		return nil, fmt.Errorf("'%s' invalid provider", provider)
	}

	var resources map[string]*tfjson.Schema

	switch ty {
	case "resource":
		resources = s.Schemas[provider].ResourceSchemas
	case "data":
		resources = s.Schemas[provider].DataSourceSchemas
	default:
		return nil, fmt.Errorf("'%s' invalid type", ty)
	}

	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, name)
	}

	sort.Strings(names)

	return names, nil
}
