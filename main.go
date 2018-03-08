package main

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

func getBDev(iIds []string) {

	//for _, i := range iIds {
	//	fmt.Println("getBdev:", i)
	//}
	for _, i := range iIds {
		// https://doco//s.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceAttribute
		// https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_instance.go

		input := &ec2.DescribeInstanceAttributeInput{
			Attribute:  aws.String("blockDeviceMapping"),
			InstanceId: aws.String(i),
		}

		result, err := svc.DescribeInstanceAttribute(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the code and
				// Message from and error
				fmt.Println(err.Error())
			}
			return
		}
		fmt.Println(result)
	} // !- for id

} // !- getBdev()

func main() {

	svc := service()
	_ = svc
	// iIds is a pointer to a slice of string
	iIds := getTags()
	fmt.Println("Type:", reflect.TypeOf(iIds))
	for _, i := range iIds {
		fmt.Println("Instance Id,", i)
		//fmt.Println("Instance e,", e)
	}
	getBDev(iIds)

}
