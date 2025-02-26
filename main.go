package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/manifoldco/promptui"
)

func containsIgnoreCase(str, substr string) bool {
	return len(str) >= len(substr) && (str[:len(substr)] == substr || containsIgnoreCase(str[1:], substr))
}

func main() {
	cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("test"))
	ec2Client := ec2.NewFromConfig(cfg)

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
					// fmt.Printf("Instance ID: %s, Name: %s\n", *instance.InstanceId, *tag.Value)
					nameToId[*tag.Value] = *instance.InstanceId
				}
			}
		}
	}

	keys := make([]string, 0, len(nameToId))
	for k := range nameToId {
		keys = append(keys, k)
	}

	instancePrompt := promptui.Select{
		Label: "Name",
		Items: keys,
		Searcher: func(input string, index int) bool {
			// Match the input to the items by checking if the item contains the input string (case-insensitive)
			item := keys[index]
			return (len(input) == 0) || (containsIgnoreCase(item, input))
		},
	}

	_, res, err := instancePrompt.Run()

	if err != nil {
		panic(err)
	}

	instanceId := nameToId[res]

	operationPrompt := promptui.Select{
		Label: "Operation",
		Items: []string{"Tunnel", "Login"},
	}

	_, op, err := operationPrompt.Run()

	if err != nil {
		panic(err)
	}

	if op == "Tunnel" {

		portPrompt := promptui.Prompt{
			Label: "Source Port",
		}

		sourcePort, err := portPrompt.Run()

		if err != nil {
			panic(err)
		}

		destPrompt := promptui.Prompt{
			Label: "Dest Port",
		}

		destPort, err := destPrompt.Run()

		if err != nil {
			panic(err)
		}

		_cmd := fmt.Sprintf(`aws ssm start-session --target %s --document-name AWS-StartPortForwardingSession --parameters '{"portNumber":["%s"], "localPortNumber":["%s"]}'  --profile test`, instanceId, sourcePort, destPort)
		println(_cmd)

		cmd := exec.Command("bash", "-c", _cmd)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Start()

		if err != nil {
			panic(err)
		}

		err = cmd.Wait()

		if err != nil {
			panic(err)
		}

	} else if op == "Login" {

		_cmd := fmt.Sprintf(`aws ssm start-session --target %s --profile test`, instanceId)
		println(_cmd)

		fmt.Printf("\033]0;%s\007", res)

		err := syscall.Exec("/bin/bash", []string{"bash", "-c", _cmd}, os.Environ())

		panic(err)

		// cmd := exec.Command("bash", "-c", _cmd)

		// cmd.Stdout = os.Stdout
		// cmd.Stderr = os.Stderr
		// cmd.Stdin = os.Stdin

		// cmd.Env = append(os.Environ(), "TERM=xterm")
		// cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// err = cmd.Run()

		// if err != nil {
		// 	panic(err)
		// }

		// err = cmd.Wait()

		// if err != nil {
		// 	panic(err)
		// }

	} else {
		println("Invalid operation!!!")
	}

}
