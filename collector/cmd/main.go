package main

import (
	"flag"
	"io/ioutil"
	"log"
	"path/filepath"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	Handler "github.com/Luismorlan/newsmux/collector/handler"
	"github.com/Luismorlan/newsmux/protocol"
	. "github.com/Luismorlan/newsmux/utils/flag"
	. "github.com/Luismorlan/newsmux/utils/log"
)

const (
	DataDir = "collector/cmd/data"
)

var (
	JobId *string
)

// init() will always be called on before the execution of main function.
func init() {
	JobId = flag.String("job_id", "", "Panoptic job id to execute")
}

func ValidatePanopticJob(job *protocol.PanopticJob) {
	if job == nil {
		log.Fatalln("job is nil")
	}
	if !job.Debug {
		log.Fatalln("must be debug PanopticJob")
	}
}

// Index all panoptic jobs in data folder, by the job id
func ParseAndIndexPanopticJobs() map[string]*protocol.PanopticJob {
	files, err := ioutil.ReadDir(DataDir)
	if err != nil {
		log.Fatalln(err)
	}

	res := []byte{}
	for _, file := range files {
		in, err := ioutil.ReadFile(filepath.Join(DataDir, file.Name()))
		if err != nil {
			log.Fatalln(err)
		}
		res = append(res, in...)
	}

	jobs := &protocol.PanopticJobs{}
	if err := prototext.Unmarshal(res, jobs); err != nil {
		log.Fatalln(err)
	}

	index := make(map[string]*protocol.PanopticJob)
	for _, job := range jobs.Jobs {
		if _, ok := index[job.JobId]; ok {
			log.Fatalln("duplicate job id in testing directory: ", job.JobId)
		}
		index[job.JobId] = job
	}

	return index
}

// This testing function will execute any config defined in //collector/cmd/data
// directory. It execute the job and publish the result into StdErrorSink.
// Example:
// go run collector/cmd/main.go -job_id "kuailansi_job"
func main() {
	ParseFlags()
	InitLogger()

	if *JobId == "" {
		log.Fatalln("job_id is required!")
	}

	index := ParseAndIndexPanopticJobs()

	job, ok := index[*JobId]
	if !ok {
		log.Fatalln("job id:", job.JobId, "is not found")
	}

	ValidatePanopticJob(job)

	log.Println("====== Panoptic Job ======\n" + proto.MarshalTextString(job))

	handler := Handler.DataCollectJobHandler{}
	if err := handler.Collect(job); err != nil {
		log.Fatalln(err)
	}

	log.Println("====== Collector Exit ======")
}
