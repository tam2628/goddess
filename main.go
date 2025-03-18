package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/manifoldco/promptui"
)

func main() {

	profiles, err := GetAllAwsProfiles()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	profile := flag.String("profile", "", "AWS Profile")

	flag.Parse()

	if *profile == "" {
		fmt.Printf("Please provide a valid AWS profile name with --profile flag. Available profiles: %s", strings.Join(profiles, ", "))
		return
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var found bool = false

	for _, p := range profiles {
		if p == *profile {
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("Invalid AWS profile. Available profiles: %s", strings.Join(profiles, ", "))
	}

	cfg, _ := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(*profile))
	ec2Client := ec2.NewFromConfig(cfg)

	instanceNameToId := GetRunningAwsInstances(ec2Client)
	names := make([]string, 0, len(instanceNameToId))

	for k := range instanceNameToId {
		names = append(names, k)
	}

	instancePrompt := promptui.Select{
		Label: "Name",
		Items: names,
		Searcher: func(input string, index int) bool {
			// Match the input to the items by checking if the item contains the input string (case-insensitive)
			item := names[index]
			return (len(input) == 0) || (ContainsIgnoreCase(item, input))
		},
	}

	_, res, err := instancePrompt.Run()

	if err != nil {
		panic(err)
	}

	instanceId := instanceNameToId[res]

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

		_cmd := fmt.Sprintf(`aws ssm start-session --target %s --document-name AWS-StartPortForwardingSession --parameters '{"portNumber":["%s"], "localPortNumber":["%s"]}'  --profile %s`, instanceId, sourcePort, destPort, *profile)
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

		_cmd := fmt.Sprintf(`aws ssm start-session --target %s --profile prod`, instanceId)
		println(_cmd)

		fmt.Printf("\033]0;%s\007", res)

		err := syscall.Exec("/bin/bash", []string{"bash", "-c", _cmd}, os.Environ())

		panic(err)

	} else {
		println("Invalid operation!!!")
	}

}
