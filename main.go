package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var svc *ec2.EC2

func service() *ec2.EC2 {

	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	os.Setenv("AWS_PROFILE", "default")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc = ec2.New(sess)

	return svc

}

//func getTags(*ec2.EC2) {
func getTags() []string {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Test"),
				Values: []*string{
					aws.String("daily"), aws.String("weekly"), aws.String("monthly")},
			},
		},
	}
	resp, err := svc.DescribeInstances(params)
	if err != nil {
		fmt.Println("error listing instances in", err.Error())
		log.Fatal(err.Error())
	}

	var iIds []string

	//for r, _ := range resp.Reservations {
	for r := range resp.Reservations {
		for _, inst := range resp.Reservations[r].Instances {
			fmt.Println("Instance Id: ", *inst.InstanceId)
			iIds = append(iIds, *inst.InstanceId)
		}
	}

	return iIds
}

func main() {

	svc := service()
	_ = svc
	// iIds is a pointer to a slice of string
	iIds := getTags()
	for i := range iIds {
		fmt.Println("Instance Id,", i)
	}

}
