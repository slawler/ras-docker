package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"gopkg.in/yaml.v2"

	"github.com/usace/wat-go-sdk/plugin"
)

const RAS_LIB_PATH = "/ras/libs:/ras/libs/mkl:/ras/libs/rhel_8"
const RAS_EXE = "/ras/v61"
const MODEL_DIR = "/sim/model"
const SCRIPT = "/app/run-model.sh"

func main() {
	var payloadFile string
	flag.StringVar(&payloadFile, "payload", "s3-key.yml", "path to s3 payload file")
	flag.Parse()

	S3Bucket := os.Getenv("AWS_BUCKET")

	fmt.Println("Reading payloadFile", payloadFile)
	payload, err := fetchPayload(S3Bucket, payloadFile)
	if err != nil {
		log.Fatal(err)
	}

	logFile := filepath.Join(MODEL_DIR, payload.Model.Name+payload.Model.Alternative+".log")
	logOutput, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer logOutput.Close()

	// Print to terminal and log file for dev
	mw := io.MultiWriter(os.Stdout, logOutput)
	log.SetOutput(mw)

	fmt.Println("Fetching Inputs...")
	_, err = fetchInputs(payload, MODEL_DIR)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Running model....")
	err = runModel(payload, MODEL_DIR)
	if err != nil {
		log.Fatal("Error running model:", err)
	}

	fmt.Println("Pushing results......")
	err = pushOutputs(payload, MODEL_DIR)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done.")
}

func fetchPayload(bucket, payloadFile string) (plugin.ModelPayload, error) {
	payload := plugin.ModelPayload{}

	svc := s3.New(session.New())
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(payloadFile),
	}

	obj, err := svc.GetObject(input)
	if err != nil {
		return payload, err
	}
	defer obj.Body.Close()

	body, err := ioutil.ReadAll(obj.Body)
	if err != nil {
		return payload, err
	}

	err = yaml.Unmarshal(body, &payload)
	if err != nil {
		return payload, err
	}

	return payload, nil
}

func fetchInputs(payload plugin.ModelPayload, localDir string) ([]string, error) {

	localFiles := make([]string, len(payload.Inputs))
	svc := s3.New(session.New())

	for i, link := range payload.Inputs {

		log.Println(i, link.ResourceInfo.Path)
		input := &s3.GetObjectInput{
			Bucket: aws.String(link.ResourceInfo.Root),
			Key:    aws.String(link.ResourceInfo.Path),
		}

		obj, err := svc.GetObject(input)
		if err != nil {
			log.Fatal("S3 Fetch Error | ", link.ResourceInfo.Path, err)
			return localFiles, err
		}
		defer obj.Body.Close()

		fileName := filepath.Base(link.ResourceInfo.Path)
		localFile := filepath.Join(localDir, fileName)

		f, err := os.OpenFile(localFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			log.Fatal("Open File Error", err)
			return localFiles, err
		}
		defer f.Close()

		_, err = io.Copy(f, obj.Body)
		if err != nil {
			log.Fatal("Write File Error", err)
			return localFiles, err
		}

		localFiles[i] = localFile

	}

	return localFiles, nil
}

func remove(slice []float64, i int) []float64 {
	return append(slice[:i], slice[i+1:]...)
}

func getPercentComplete(s string, cv *[]float64) bool {
	logEntry := strings.Split(s, "=")
	pctComplete := strings.TrimSpace(logEntry[1])
	if num, err := strconv.ParseFloat(pctComplete, 32); err == nil {
		for i, val := range *cv {
			if math.Abs(val-num) < 0.01 {
				*cv = remove(*cv, i)
				return true
			}
		}
	}
	return false
}

func rasPctLog(startLogging *int, message string, checkValues *[]float64) {

	// RAS hack to only print progress every 10%
	if strings.Contains(message, "LABEL= Unsteady Flow Computations") {
		// trigger logging
		*startLogging += 1
	}

	if strings.Contains(message, "LABEL= Unsteady Flow Warmup") {
		// trigger logging
		*startLogging -= 1
	}

	if *startLogging > 0 && strings.Contains(message, "PROGRESS") {
		msg := getPercentComplete(message, checkValues)

		if msg == true {
			log.Printf(message)

		}
	}
}

func runModel(payload plugin.ModelPayload, localDir string) error {
	fmt.Println("Run Args: ", SCRIPT, localDir, payload.Model.Name)

	cmd := exec.Command(SCRIPT, localDir, payload.Model.Name)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println(err)
		return err
	}

	if err := cmd.Start(); err != nil {
		log.Println(err)
		return err
	}

	in := bufio.NewScanner(stdout)

	// Logging placeholder
	startLogging := 0
	checkValues := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}

	for in.Scan() {
		message := in.Text()
		rasPctLog(&startLogging, message, &checkValues)
	}

	if err := in.Err(); err != nil {
		log.Println(err)
		return err
	}

	// Placeholder for renaming the output p*/hdf following successful sim
	var localFiles []string
	err = filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		localFiles = append(localFiles, path)
		return err
	})

	if err != nil {
		log.Println(err)
		return err
	}

	for _, file := range localFiles {
		if strings.Contains(file, ".tmp") {
			err := os.Rename(file, strings.Replace(file, ".tmp", "", 1))
			if err != nil {
				log.Println(err)
				return err
			}
		}
	}

	return nil
}

func pushOutputs(payload plugin.ModelPayload, localDir string) error {

	svc := s3.New(session.New())

	for _, link := range payload.Outputs {

		fileName := filepath.Base(link.ResourceInfo.Path)
		localFile := filepath.Join(localDir, fileName)

		file, err := os.Open(localFile)
		if err != nil {
			log.Println(err)
			return err
		}

		fileInfo, _ := file.Stat()
		size := fileInfo.Size()
		buffer := make([]byte, size)
		file.Read(buffer)

		_, err = svc.PutObject(&s3.PutObjectInput{
			Bucket:        aws.String(link.ResourceInfo.Root),
			Key:           aws.String(link.ResourceInfo.Path),
			Body:          bytes.NewReader(buffer),
			ContentLength: aws.Int64(size),
			ContentType:   aws.String(http.DetectContentType(buffer)),
		})
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
