package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils/flag"
	. "github.com/Luismorlan/newsmux/utils/log"
)

const (
	InputDir   = "scripts/prepare_http_header"
	InputFile  = "input.txt"
	OutputFile = "output.txt"
)

var (
	JobId *string
)

func ValidatePanopticJob(job *protocol.PanopticJob) {
	if job == nil {
		log.Fatalln("job is nil")
	}
	if !job.Debug {
		log.Fatalln("must be debug PanopticJob")
	}
}

// This script will generate the "headers" part in the panoptic job config
// Oncall can go to the destination page, find the xhr request in browser console
// Click "Copy -> Copy as fetch"
// Then put the intput in "input.txt"
// headers will be formated into "output.txt"
func main() {
	ParseFlags()
	InitLogger()

	in, err := ioutil.ReadFile(filepath.Join(InputDir, InputFile))
	if err != nil {
		log.Fatalln(err)
	}
	trimed := strings.Replace(string(in), "\n", "", -1)
	r := regexp.MustCompile(`\"headers\"\: (\{.*?\})`)
	allMatched := r.FindStringSubmatch(trimed)
	if len(allMatched) < 2 {
		log.Fatal("No header matched")
	}
	jsonStr := allMatched[1]

	var f interface{}
	err = json.Unmarshal([]byte(jsonStr), &f)
	if err != nil {
		fmt.Println("Error parsing JSON: ", err)
	}

	itemsMap := f.(map[string]interface{})
	var kvs []*protocol.KeyValuePair
	for k, v := range itemsMap {
		kvs = append(kvs, &protocol.KeyValuePair{Key: k, Value: v.(string)})
	}

	file, err := os.Create(filepath.Join(InputDir, OutputFile))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(file, "header_params: [\n")
	for ind, v := range kvs {
		escaped := strings.Replace(v.Value, `"`, `\"`, -1)
		fmt.Fprintf(file, "\t{key: \"%s\", value: \"%s\"}", v.Key, escaped)
		if ind != len(kvs)-1 {
			fmt.Fprintf(file, ",")
		}
		fmt.Fprintf(file, "\n")
	}
	fmt.Fprintf(file, "]\n")
}
