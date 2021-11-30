package main

import (
	"asialoop.de/deploy-eb-go/ebclient"
	"context"
	"flag"
	"github.com/aws/aws-sdk-go-v2/config"
	"log"
)

var applicationName = flag.String("app", "", "EB application name")
var bucket = flag.String("bucket", "", "S3 bucket to store application deployment artifacts")
var env = flag.String("env", "", "Environment prefix")
var version = flag.String("version", "", "Application version")
var file = flag.String("file", "", "Deployment artifact")

func main() {
	flag.Parse()

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}

	ebClient := ebclient.NewEBClient(cfg, *applicationName, *bucket)
	ebClient.UpdateEnv(&ebclient.UpdateEnvInput{
		EnvPrefix: *env,
		Version:   *version,
		FilePath:  *file,
	})
}
