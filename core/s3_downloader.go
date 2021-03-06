package core

import (
	"os"
	"path"

	"code.cloudfoundry.org/lager"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/snickers/snickers/db"
	"github.com/snickers/snickers/types"
)

// S3Download downloads the file from S3 bucket. Job Source should be
// in format: http://AWSKEY:AWSSECRET@BUCKET.s3.amazonaws.com/OBJECT
func S3Download(logger lager.Logger, dbInstance db.Storage, jobID string) error {
	log := logger.Session("s3-download")
	log.Info("start", lager.Data{"job": jobID})
	defer log.Info("finished")

	job, err := dbInstance.RetrieveJob(jobID)
	if err != nil {
		log.Error("retrieving-job", err)
		return err
	}

	job.LocalSource = GetLocalSourcePath(job.ID) + path.Base(job.Source)
	job.LocalDestination = GetLocalDestination(dbInstance, jobID)
	job.Destination = GetOutputFilename(dbInstance, jobID)
	job.Status = types.JobDownloading
	job.Details = "0%"
	dbInstance.UpdateJob(job.ID, job)

	file, err := os.Open(job.LocalDestination)
	if err != nil {
		return err
	}

	err = SetAWSCredentials(job.Source)
	if err != nil {
		return err
	}

	bucket, err := GetAWSBucket(job.Source)
	if err != nil {
		return err
	}

	key, err := GetAWSKey(job.Source)
	if err != nil {
		return err
	}

	downloader := s3manager.NewDownloader(session.New(&aws.Config{Region: aws.String("us-east-1")}))
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
	if err != nil {
		return err
	}
	return nil
}
