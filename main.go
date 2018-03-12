package main

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/mitchellh/mapstructure"
)

// General refs
//   https://doco//s.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceAttribute
//   https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_instance.go

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

// Get list of tagged instances
// !+getTaggedInstances()
func getTaggedInstances(t string) []*ec2.Instance {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:" + t),
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

	var taggedInstances []*ec2.Instance

	// Reservations can have one or more taggedInstances,
	// so we need to loop through it twice, first getting a Reservation
	// then getting instances for the reservation.
	for r := range resp.Reservations {
		for _, inst := range resp.Reservations[r].Instances {
			if inst != nil {
				taggedInstances = append(taggedInstances, inst)
			}
		}
	}
	return taggedInstances
} // !-getTaggedInstances()

// Get block devices from instance
//!+readBlockDeviceFromInstance()
func readBlockDeviceFromInstance(instance *ec2.Instance) (map[string]interface{}, error) {

	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil

	instanceBlockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs != nil {
			instanceBlockDevices[*bd.Ebs.VolumeId] = bd
			//fmt.Println("bd:", bd)
		}
	}

	if len(instanceBlockDevices) == 0 {
		return nil, nil
	}

	volIDs := make([]*string, 0, len(instanceBlockDevices))
	for volID := range instanceBlockDevices {
		volIDs = append(volIDs, aws.String(volID))
		//fmt.Println("volID:", volID)
	}

	// Call DescribeVolumes to get vol size
	volResp, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{
		VolumeIds: volIDs,
	})
	if err != nil {
		return nil, err
	}

	for _, vol := range volResp.Volumes {
		instanceBd := instanceBlockDevices[*vol.VolumeId]
		bd := make(map[string]interface{})

		bd["volume_id"] = *vol.VolumeId

		if instanceBd.Ebs != nil && instanceBd.Ebs.DeleteOnTermination != nil {
			bd["delete_on_termination"] = *instanceBd.Ebs.DeleteOnTermination
		}
		if vol.Size != nil {
			bd["volume_size"] = *vol.Size
		}
		if vol.VolumeType != nil {
			bd["volume_type"] = *vol.VolumeType
		}
		if vol.Iops != nil {
			bd["iops"] = *vol.Iops
		}
		if blockDeviceIsRoot(instanceBd, instance) {
			blockDevices["root"] = bd
		} else {

			if instanceBd.DeviceName != nil {
				bd["device_name"] = *instanceBd.DeviceName
			}
			if vol.Encrypted != nil {
				bd["encrypted"] = *vol.Encrypted
			}
			if vol.SnapshotId != nil {
				bd["snapshot_id"] = *vol.SnapshotId
			}

			blockDevices["ebs"] = append(blockDevices["ebs"].([]map[string]interface{}), bd)
		}
	}

	return blockDevices, nil

} //!- readBlockDeviceFromInstance()

//!+blockDeviceIsRoot()
func blockDeviceIsRoot(bd *ec2.InstanceBlockDeviceMapping, instance *ec2.Instance) bool {
	return bd.DeviceName != nil &&
		instance.RootDeviceName != nil &&
		*bd.DeviceName == *instance.RootDeviceName
} // !-blockDeviceIsRoot()

//!+main
func main() {

	svc := service()
	_ = svc
	// instances is a pointer to a slice of string
	tag := "Test"
	instances := getTaggedInstances(tag)
	//fmt.Println("Type of instances:", reflect.TypeOf(instances))

	for _, i := range instances {
		//fmt.Println("Type of i:", reflect.TypeOf(i))
		fmt.Println("i.InstanceId:", *i.InstanceId)

		// mitchellh/mapstructure
		fmt.Println("--------------------")
		var ec2Instance *ec2.Instance
		err := mapstructure.Decode(i, &ec2Instance)
		if err != nil {
			panic(err)
		}
		//fmt.Printf("ec2Instance.Decode: %v\n", ec2Instance)
		fmt.Printf("ec2Instance.InstanceId: %v\n", *ec2Instance.InstanceId)
		//fmt.Printf("ec2Instance.Tags: %v\n", ec2Instance.Tags)
		fmt.Printf("type of ec2Instance.Tags: %v\n", reflect.TypeOf(ec2Instance.Tags))
		for _, tag := range ec2Instance.Tags {
			//fmt.Println("type of tag:", reflect.TypeOf(tag))
			//fmt.Println("tag:", tag)
			fmt.Printf("Tag: %v => %v\n", *tag.Key, *tag.Value)
		}

		//fmt.Printf("ec2Instance.BlockDeviceMappings: %v\n", *ec2Instance.BlockDeviceMappings)
		//fmt.Printf("ec2Instance.BlockDeviceMappings: %v\n", ec2Instance.BlockDeviceMappings)
		for _, bdm := range ec2Instance.BlockDeviceMappings {
			//fmt.Println("bdm:", bdm)
			fmt.Println("bdm.DeviceName:", *bdm.DeviceName)
			//fmt.Println("bdm.Ebs:", *bdm.Ebs)
			//fmt.Println("type of bdm.Ebs:", reflect.TypeOf(*bdm.Ebs))
			fmt.Println("bdm.Ebs.VolumeId:", *bdm.Ebs.VolumeId)
			//fmt.Println("bdm.Ebs.Tags:", *bdm.Ebs.Tags)

		}

		//fmt.Println("Type of bdevs:", reflect.TypeOf(bdevs))
		fmt.Println("--------------------")

	}
} //!-main
