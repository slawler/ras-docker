package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"gopkg.in/yaml.v2"
)

const RAS_LIB_PATH = "/ras/libs:/ras/libs/mkl:/ras/libs/rhel_8"
const RAS_EXE = "/ras/v61"
const MODEL_DIR = "/sim/model"
const SCRIPT = "/app/run-model.sh"

func main() {
	var payloadFile string
	flag.StringVar(&payloadFile, "f", "s3-key.yml", "path to s3 payload file")

	var s3Endpoint string
	flag.StringVar(&s3Endpoint, "m", "", "S3Endpoint for mocking")

	flag.Parse()

	S3Bucket := os.Getenv("AWS_BUCKET")

	fmt.Println("Reading payloadFile", payloadFile)
	payload, err := fetchPayload(S3Bucket, payloadFile, s3Endpoint)
	if err != nil {
		log.Fatal(err)
	}

	logFile := filepath.Join(MODEL_DIR, payload.ModelConfiguration.ModelName+".log")
	logOutput, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer logOutput.Close()

	// Print to terminal and log file for dev
	mw := io.MultiWriter(os.Stdout, logOutput)
	log.SetOutput(mw)

	fmt.Println("Fetching Inputs...")
	_, err = fetchInputs(payload, MODEL_DIR, s3Endpoint)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Running model....")
	err = runModel(payload, MODEL_DIR)
	if err != nil {
		log.Fatal("Error running model:", err)
	}

	fmt.Println("Pushing results......")
	err = pushOutputs(payload, MODEL_DIR, s3Endpoint)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done.")
}

func awsS3Session(s3Endpoint string) (*s3.S3, error) {
	var err error
	keyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	creds := credentials.NewStaticCredentials(keyID, secretKey, "")
	cfg := aws.NewConfig().WithRegion(region).WithCredentials(creds)

	if s3Endpoint != "" {
		cfg.WithDisableSSL(true)
		cfg.WithS3ForcePathStyle(true)
		cfg.WithEndpoint(s3Endpoint)
		fmt.Println("Using Mock environment", s3Endpoint)
		session, err := session.NewSession(cfg)
		return s3.New(session), err
	}

	session, err := session.NewSession(cfg)
	return s3.New(session), err
}

func fetchPayload(bucket, payloadFile, s3Endpoint string) (Payload, error) {
	payload := Payload{}

	svc, err := awsS3Session(s3Endpoint)
	if err != nil {
		return payload, err
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(payloadFile),
	}

	// fmt.Println(bucket, payloadFile)

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

func fetchInputs(payload Payload, localDir, s3Endpoint string) ([]string, error) {

	localFiles := make([]string, len(payload.ModelLinks.LinkedInputs))
	svc, err := awsS3Session(s3Endpoint)
	if err != nil {
		return localFiles, err
	}

	for i, link := range payload.ModelLinks.LinkedInputs {

		log.Println(i, link.ResourceInfo.Fragment)
		input := &s3.GetObjectInput{
			Bucket: aws.String(link.ResourceInfo.Authority),
			Key:    aws.String(link.ResourceInfo.Fragment),
		}

		obj, err := svc.GetObject(input)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
			return localFiles, err
		}
		defer obj.Body.Close()

		fileName := filepath.Base(link.ResourceInfo.Fragment)
		localFile := filepath.Join(localDir, fileName)

		f, err := os.OpenFile(localFile, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
			return localFiles, err
		}
		defer f.Close()

		_, err = io.Copy(f, obj.Body)
		if err != nil {
			log.Fatalf("ERROR: %s", err)
			return localFiles, err
		}

		localFiles[i] = localFile

	}

	return localFiles, nil
}

func runModel(payload Payload, localDir string) error {
	fmt.Println("Run Args: ", SCRIPT, localDir, payload.ModelConfiguration.ModelName)

	cmd := exec.Command(SCRIPT, localDir, payload.ModelConfiguration.ModelName)
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

	for in.Scan() {
		log.Printf(in.Text())
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

func pushOutputs(payload Payload, localDir, s3Endpoint string) error {
	svc, err := awsS3Session(s3Endpoint)
	if err != nil {
		return err
	}

	for _, link := range payload.ModelLinks.RequiredOutputs {

		fileName := filepath.Base(link.ResourceInfo.Fragment)
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
			Bucket:        aws.String(link.ResourceInfo.Authority),
			Key:           aws.String(link.ResourceInfo.Fragment),
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

// Placeholder for storing payload instructions while specification is in progress
//  Generated using: https://zhwt.github.io/yaml-to-go/

type Payload struct {
	ModelConfiguration ModelConfiguration `yaml:"model_configuration"`
	ModelLinks         ModelLinks         `yaml:"model_links"`
}

type ModelConfiguration struct {
	ModelName string `yaml:"model_name"`
}

type ResourceInfo struct {
	Scheme    string `yaml:"scheme"`
	Authority string `yaml:"authority"`
	Fragment  string `yaml:"fragment"`
}

type LinkedInputs struct {
	Name         string       `yaml:"name"`
	Format       string       `yaml:"format"`
	ResourceInfo ResourceInfo `yaml:"resource_info"`
}

type RequiredOutputs struct {
	Name         string       `yaml:"name"`
	Format       string       `yaml:"format"`
	ResourceInfo ResourceInfo `yaml:"resource_info"`
}

type ModelLinks struct {
	LinkedInputs    []LinkedInputs    `yaml:"linked_inputs"`
	RequiredOutputs []RequiredOutputs `yaml:"required_outputs"`
}
