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

// Get list of taged instances
func getTaggedInstances() []string {
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

	var instances []string

	//for r, _ := range resp.Reservations {
	for r := range resp.Reservations {
		for _, inst := range resp.Reservations[r].Instances {
			fmt.Println("Instance Id: ", *inst.InstanceId)
			instances = append(instances, *inst.InstanceId)
		}
	}

	return instances
}

// Get block devices from instance
//func readBlockDeviceFromInstance(instances []string) {
func readBlockDeviceFromInstance(instance string) {

	// BlockDevice containes all fields necessary
	// to take a snapshot
	type BlockDevice struct {
		InstanceId string
		DeviceName string
		VolumeId   string
	}

	// General refs%
	//   https://doco//s.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceAttribute
	//   https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_instance.go

	//InstancId := id
	fmt.Println("instance", instance)

	bd := BlockDevice{
		InstanceId: instance,
	}

	input := &ec2.DescribeInstanceAttributeInput{
		Attribute:  aws.String("blockDeviceMapping"),
		InstanceId: aws.String(instance),
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
	fmt.Println("Type of result.BockDeviceMappings.BlockDeviceMappings:", reflect.TypeOf(result.BlockDeviceMappings))
	//fmt.Println(result)
	fmt.Println(result.BlockDeviceMappings)
	//fmt.Println(result.BlockDeviceMappings.DeviceName)

	//var instanceBlockDevices []*ec2.InstanceBlockDeviceMapping
	//instanceBlockDevices = result.BlockDeviceMappings
	instanceBlockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
	for i, bd := range instanceBlockDevices {
		if bd.Ebs != nil {
			fmt.Println("index:", i, "Block Device:", bd)
		}
		//fmt.Println("index:", i)
		////fmt.Println(bd)
	}

	// make a slice of BlockDevice and add all volumes to be snapshotted
	fmt.Println("InstanceId:", bd.InstanceId)
	//fmt.Println("InstanceId:", InstanceId)
	fmt.Println("DeviceName:")
	fmt.Println("VolumeId:")

	//} // !- for id

} // !- readBlockDeviceFromInstance()

func main() {

	svc := service()
	_ = svc
	// instances is a pointer to a slice of string
	instances := getTaggedInstances()
	fmt.Println("Type:", reflect.TypeOf(instances))
	fmt.Println("instances:", instances)
	for _, i := range instances {
		fmt.Println("Instance Id", i)
		readBlockDeviceFromInstance(i)
	}
	//readBlockDeviceFromInstance(instances)

}
