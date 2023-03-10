package runners

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type OGCRunner struct {
	Bucket      string `json:"bucket"`
	PayloadFile string `json:"payload_file"`
	LocalDir    string `json:"local_dir"`
	Payload     `json:"payload"`
}

type Payload struct {
	ModelName string    `json:"model_name"`
	Inputs    []Inputs  `json:"inputs"`
	Outputs   []Outputs `json:"outputs"`
}

type Inputs struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

type Outputs struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

func (r *OGCRunner) ModelName() string {
	return r.Payload.ModelName
}

func (r *OGCRunner) PrepRun() error {

	err := r.fetchPayload()
	if err != nil {
		return err
	}

	localFiles := make([]string, len(r.Payload.Inputs))
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		return err
	}

	svc := s3.New(sess)

	for i, link := range r.Payload.Inputs {
		fmt.Println(i, link)

		input := &s3.GetObjectInput{
			Bucket: aws.String(r.Bucket),
			Key:    aws.String(link.Href),
		}

		obj, err := svc.GetObject(input)
		if err != nil {
			fmt.Println("S3 Fetch Error | ", link.Href, err)
			return err
		}
		defer obj.Body.Close()

		fileName := filepath.Base(link.Href)
		localFile := filepath.Join(r.LocalDir, fileName)

		f, err := os.OpenFile(localFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			fmt.Println("Open File Error", err)
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, obj.Body)
		if err != nil {
			fmt.Println("Write File Error", err)
			return err
		}

		localFiles[i] = localFile

		msg := fmt.Sprintf("downloaded s3://%s/%s to %s", r.Bucket, link.Href, localFile)
		fmt.Println(msg)

	}

	return nil
}

func (r *OGCRunner) Run() error {

	cmd := exec.Command("/app/run-model.sh", "/sim/model", r.ModelName())
	cmd.Dir = r.LocalDir
	msg := fmt.Sprintf("running model from directory '%s' with args: [ %s ]", r.LocalDir, strings.Join(cmd.Args, ", "))
	fmt.Println(msg)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return err
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		return err
	}

	in := bufio.NewScanner(stdout)

	// Logging placeholder
	startLogging := 0
	checkValues := map[string]float64{
		"10%":  0.1,
		"20%":  0.2,
		"30%":  0.3,
		"40%":  0.4,
		"50%":  0.5,
		"60%":  0.6,
		"70%":  0.7,
		"80%":  0.8,
		"90%":  0.9,
		"100%": 1.0,
	}

	computeLog := filepath.Join(r.LocalDir, r.ModelName()+".log")
	f, err := os.OpenFile(computeLog, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer f.Close()

	for in.Scan() {
		message := in.Text()
		rasPctLog(&startLogging, message, &checkValues)
		_, err := f.WriteString(message + "\n")
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	if err := in.Err(); err != nil {
		fmt.Println(err)
		return err
	}

	// Rename the output p*/hdf following successful sim
	err = filepath.Walk(r.LocalDir, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, ".tmp") {
			err := os.Rename(path, strings.Replace(path, ".tmp", "", 1))
			if err != nil {
				fmt.Println(err)
				return err
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (r *OGCRunner) CopyOutputs() error {

	sess, err := session.NewSession()
	if err != nil {
		fmt.Println(err)
		return err
	}
	svc := s3.New(sess)

	for _, link := range r.Payload.Outputs {

		fileName := filepath.Base(link.Href)
		localFile := filepath.Join(r.LocalDir, fileName)

		file, err := os.Open(localFile)
		if err != nil {
			fmt.Println(err)
			return err
		}

		fileInfo, err := file.Stat()
		if err != nil {
			fmt.Println(err)
			return err
		}
		size := fileInfo.Size()

		contentType := mime.TypeByExtension(filepath.Ext(localFile))
		if contentType == "" {
			if filepath.Ext(localFile) == ".log" { // this one is not caught for some reason
				contentType = "text/plain"
			} else {
				contentType = "application/octet-stream"
			}
		}

		_, err = svc.PutObject(&s3.PutObjectInput{
			Bucket:        aws.String(link.Href),
			Key:           aws.String(link.Href),
			Body:          file,
			ContentLength: aws.Int64(size),
			ContentType:   aws.String(contentType),
		})
		if err != nil {
			fmt.Println(err)
			return err
		}

		msg := fmt.Sprintf("uploaded %s to s3://%s/%s", localFile, r.Bucket, link.Href)
		fmt.Println(msg)

	}
	return nil
}

func (r *OGCRunner) fetchPayload() error {
	var payload Payload

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		return err
	}

	svc := s3.New(sess)
	input := &s3.GetObjectInput{
		Bucket: aws.String(r.Bucket),
		Key:    aws.String(r.PayloadFile),
	}

	obj, err := svc.GetObject(input)
	if err != nil {
		return err
	}
	defer obj.Body.Close()

	body, err := ioutil.ReadAll(obj.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &payload)
	if err != nil {
		return err
	}

	r.Payload = payload

	return nil
}
