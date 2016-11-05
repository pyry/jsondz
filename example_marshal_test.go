package jsondz

import (
	"fmt"
	"strings"
)

var exampleJSON = []string{
	`{"Region":"eu-central-1","Bucket":"com.foo.bar"}`,
	`{"Up":true}`,
}

func Example() {
	for _, j := range exampleJSON {
		w, _ := Unmarshal([]byte(j), caseWorker{}, complexSetupWorker{})
		fmt.Println(w.(Worker).Work("Hello World!"))
	}
	// Output:
	// Processed work Hello World! at eu-central-1-com.foo.bar region-bucket
}

// Worker interface
type Worker interface {
	Work(work string) string
}

type caseWorker struct {
	Up bool
}

func (u caseWorker) Work(work string) string {
	if u.Up {
		return strings.ToUpper(work)
	}
	return strings.ToLower(work)
}

type complexConfig struct {
	FooRegion string `json:"Region"`
	BarBucket string `json:"Bucket"`
}

type complexSetupWorker struct {
	c complexConfig
}

func (c complexSetupWorker) Work(work string) string {
	return fmt.Sprintf(
		"Processed work %s at %s-%s region-bucket",
		work, c.c.FooRegion,
		c.c.BarBucket,
	)
}

// New returns new instance of complexSetupWorker
func (c complexSetupWorker) New(conf complexConfig) *complexSetupWorker {
	// Do compex setup...
	return &complexSetupWorker{conf}
}
