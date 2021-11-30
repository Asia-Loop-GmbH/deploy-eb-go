package ebclient

import (
	"asialoop.de/deploy-eb-go/s3client"
	"asialoop.de/deploy-eb-go/wait"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk/types"
	"github.com/hashicorp/go-uuid"
	"log"
	"strings"
	"time"
)

type EBClient struct {
	client          *elasticbeanstalk.Client
	s3Client        *s3client.S3Client
	bucket          string
	applicationName string
}

func NewEBClient(cfg aws.Config, applicationName, bucket string) *EBClient {
	return &EBClient{
		client:          elasticbeanstalk.NewFromConfig(cfg),
		s3Client:        s3client.NewS3Client(cfg),
		bucket:          bucket,
		applicationName: applicationName,
	}
}

type UpdateEnvInput struct {
	EnvPrefix string
	Version   string
	FilePath  string
}

func (ebClient *EBClient) UpdateEnv(options *UpdateEnvInput) {
	log.Printf("update environment: %v", options)
	ebClient.createNewVersion(options.FilePath, options.Version, options.EnvPrefix)
	ebClient.deployNewVersion(options.Version, options.EnvPrefix)
	log.Printf("environment updated: %v", options)
}

func (ebClient *EBClient) createNewVersion(filePath, version, envPrefix string) {
	log.Printf("create version '%s' for environment '%s***' from '%s'", version, envPrefix, filePath)
	key := fmt.Sprintf("%s/%s.zip", envPrefix, version)

	ebClient.s3Client.Upload(filePath, ebClient.bucket, key)

	_, err := ebClient.client.CreateApplicationVersion(context.TODO(), &elasticbeanstalk.CreateApplicationVersionInput{
		ApplicationName: &ebClient.applicationName,
		VersionLabel:    &version,
		SourceBundle: &types.S3Location{
			S3Bucket: &ebClient.bucket,
			S3Key:    &key,
		},
	})

	if err != nil {
		panic(err)
	}

	log.Printf("version '%s' created for environment '%s***", version, envPrefix)
}

func (ebClient *EBClient) deployNewVersion(version, envPrefix string) {
	log.Printf("deploy version '%s' to environment '%s***'", version, envPrefix)
	currentEnv := ebClient.findEnvironment(envPrefix)
	ebClient.updateConfiguration(currentEnv, envPrefix)
	newName := fmt.Sprintf("%s-%s", envPrefix, envSuffix())
	ebClient.createNewEnv(envPrefix, newName, version)
	ebClient.switchUrls(*currentEnv.EnvironmentName, newName)
	ebClient.deleteEnv(*currentEnv.EnvironmentName)
	log.Printf("version '%s' deployed to '%s***'", version, envPrefix)
}

func (ebClient *EBClient) deleteEnv(name string) {
	log.Printf("delete environment '%s'", name)
	_, err := ebClient.client.TerminateEnvironment(context.TODO(), &elasticbeanstalk.TerminateEnvironmentInput{
		EnvironmentName: &name,
	})
	if err != nil {
		panic(err)
	}
	ebClient.isEnvTerminated(name)
	log.Printf("environment '%s' deleted", name)
}

func (ebClient *EBClient) switchUrls(sourceName, destName string) {
	log.Printf("switch urls '%s' and '%s'", sourceName, destName)
	_, err := ebClient.client.SwapEnvironmentCNAMEs(context.TODO(), &elasticbeanstalk.SwapEnvironmentCNAMEsInput{
		SourceEnvironmentName:      &sourceName,
		DestinationEnvironmentName: &destName,
	})
	if err != nil {
		panic(err)
	}
	ebClient.waitEnvReady(sourceName)
	ebClient.waitEnvReady(destName)
	log.Println("switch urls done")
}

func (ebClient *EBClient) createNewEnv(envPrefix, newName, version string) {
	log.Printf("create new environment '%s'", newName)
	_, err := ebClient.client.CreateEnvironment(context.TODO(), &elasticbeanstalk.CreateEnvironmentInput{
		ApplicationName: &ebClient.applicationName,
		EnvironmentName: &newName,
		CNAMEPrefix:     &newName,
		VersionLabel:    &version,
		TemplateName:    &envPrefix,
	})
	if err != nil {
		panic(err)
	}

	ebClient.waitEnvReady(newName)

	log.Printf("environment '%s' created", newName)
}

func (ebClient *EBClient) waitEnvReady(name string) {
	log.Printf("wait environment ready of '%s'", name)
	wait.Wait(func() bool {
		return ebClient.isEnvReady(name)
	}, time.Second*15, 50)
	log.Printf("environment '%s' is ready", name)
}

func (ebClient *EBClient) isEnvReady(name string) bool {
	return ebClient.isEnvStatus(name, types.EnvironmentStatusReady)
}

func (ebClient *EBClient) isEnvTerminated(name string) bool {
	return ebClient.isEnvStatus(name, types.EnvironmentStatusTerminated)
}

func (ebClient *EBClient) isEnvStatus(name string, status types.EnvironmentStatus) bool {
	log.Printf("check environment status '%s' of '%s'", status, name)
	result, err := ebClient.client.DescribeEnvironments(context.TODO(), &elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName:  &ebClient.applicationName,
		EnvironmentNames: []string{name},
	})
	if err != nil {
		panic(err)
	}
	envs := result.Environments

	if len(envs) != 1 {
		log.Panicf("invalid amount of found environments: %v", envs)
	}

	return envs[0].Status == status
}

func (ebClient *EBClient) updateConfiguration(env types.EnvironmentDescription, envPrefix string) {
	log.Printf("update configuration '%s' using current configuration of '%s'", envPrefix, *env.EnvironmentName)
	log.Printf("delete configuration '%s'", envPrefix)
	_, err := ebClient.client.DeleteConfigurationTemplate(context.TODO(), &elasticbeanstalk.DeleteConfigurationTemplateInput{
		ApplicationName: &ebClient.applicationName,
		TemplateName:    &envPrefix,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("configuration '%s' deleted", envPrefix)
	log.Printf("create configuration '%s'", envPrefix)
	_, err = ebClient.client.CreateConfigurationTemplate(context.TODO(), &elasticbeanstalk.CreateConfigurationTemplateInput{
		ApplicationName: &ebClient.applicationName,
		TemplateName:    &envPrefix,
		EnvironmentId:   env.EnvironmentId,
	})
	if err != nil {
		panic(err)
	}
	log.Printf("configuration '%s' updated", envPrefix)
}

func (ebClient *EBClient) findEnvironment(envPrefix string) types.EnvironmentDescription {
	log.Printf("find environment by prefix '%s'", envPrefix)
	envs, err := ebClient.client.DescribeEnvironments(context.TODO(), &elasticbeanstalk.DescribeEnvironmentsInput{
		ApplicationName: &ebClient.applicationName,
	})
	if err != nil {
		panic(err)
	}

	found := make([]types.EnvironmentDescription, 0)

	for _, env := range envs.Environments {
		if env.Status == types.EnvironmentStatusReady && strings.HasPrefix(*env.EnvironmentName, envPrefix) {
			log.Printf("found %s", *env.EnvironmentName)
			found = append(found, env)
		}
	}

	if len(found) == 0 {
		log.Panicf("no environment with prefix '%s' has status 'Ready'", envPrefix)
	}

	if len(found) > 1 {
		log.Panicf("more than one environments with prefix '%s' that have status 'Ready': %v", envPrefix, found)
	}

	log.Printf("environment '%s' found", *found[0].EnvironmentName)

	return found[0]
}

func envSuffix() string {
	bytes, err := uuid.GenerateRandomBytes(4)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
