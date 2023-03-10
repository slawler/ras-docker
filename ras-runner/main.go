package main

import (
	"app/runners"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/labstack/gommon/log"
)

const (
	RAS_LIB_PATH = "/ras/libs:/ras/libs/mkl:/ras/libs/rhel_8"
	RAS_EXE      = "/ras/v61"
	MODEL_DIR    = "/sim/model"
	SCRIPT       = "/app/run-model.sh"
)

type Runner interface {
	ModelName() string
	PrepRun() error
	Run() error
	CopyOutputs() error
}

func main() {
	var runnerType string
	var payloadFile string
	var r Runner
	var err error

	flag.StringVar(&runnerType, "r", "ogc", "runner to use, currently support wat or ogc")
	flag.StringVar(&payloadFile, "f", "ogc-payloads/example.json", "path to s3 payload file")
	flag.Parse()

	switch runnerType {
	case "ogc":
		r = &runners.OGCRunner{PayloadFile: payloadFile, LocalDir: MODEL_DIR, Bucket: "cloud-wat-dev"}
		if err != nil {
			fmt.Println("Error running model:", err)
		}
	default:
		fmt.Println(err)
		return
	}

	err = r.PrepRun()
	if err != nil {
		fmt.Println(err)
		return
	}

	modelName := r.ModelName()
	logFile := filepath.Join(MODEL_DIR, modelName+".log")
	logOutput, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer logOutput.Close()

	// Print to terminal and log file for dev
	mw := io.MultiWriter(os.Stdout, logOutput)
	log.SetOutput(mw)

	fmt.Println("Running model....")
	err = r.Run()
	if err != nil {
		fmt.Println("Error running model:", err)
		return
	}

	fmt.Println("Pushing results......")
	err = r.CopyOutputs()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Done")
}
