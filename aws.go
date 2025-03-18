package main

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func GetAllAwsProfiles() ([]string, error) {
	profiles := make([]string, 0)

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	files := []string{
		filepath.Join(home, ".aws", "config"),
		filepath.Join(home, ".aws", "credentials"),
	}

	for _, file := range files {

		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				section := strings.Trim(line, "[]")
				section = strings.Trim(section, " ")
				if !strings.HasPrefix(section, "profile") {
					continue
				}

				profile := strings.TrimPrefix(section, "profile ")
				profiles = append(profiles, profile)
			}
		}

		defer f.Close()
	}

	return profiles, nil
}

func GetRunningAwsInstances(ec2Client *ec2.Client) map[string]string {

	var allInstancesEC2 []types.Instance
	var nameToId map[string]string = make(map[string]string)

	var INSTANCE_STATE_KEY string = "instance-state-name"

	input := ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   &INSTANCE_STATE_KEY,
				Values: []string{"running"},
			},
		},
	}

	paginator := ec2.NewDescribeInstancesPaginator(ec2Client, &input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}

		// Append instances from each page
		for _, r := range page.Reservations {
			allInstancesEC2 = append(allInstancesEC2, r.Instances...)
		}
	}

	for _, instance := range allInstancesEC2 {
		if instance.Tags != nil {
			for _, tag := range instance.Tags {
				if *tag.Key == "Name" {
					nameToId[*tag.Value] = *instance.InstanceId
				}
			}
		}
	}

	return nameToId
}
