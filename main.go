package main

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
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

//type taggedInstance struct {
//	instanceId string
//	instance   *ec2.Instance
//}

//func getTaggedInstances() []string {
//func getTaggedInstances(t string) []*ec2.Instance {
//func getTaggedInstances(t string) []taggedInstance {
//func getTaggedInstances(t string) map[string]*ec2.Instance {
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
	//fmt.Println("resp resp:", resp)
	////fmt.Println("resp:", reflect.TypeOf(resp))
	if err != nil {
		fmt.Println("error listing instances in", err.Error())
		log.Fatal(err.Error())
	}

	//var instances []string
	//var instances []*ec2.Instance
	//var instances []taggedInstance

	//taggedInstances := make(map[string]*ec2.Instance)
	var taggedInstances []*ec2.Instance

	//for r, _ := range resp.Reservations {
	for r := range resp.Reservations {
		fmt.Println("resp.Reservations:", reflect.TypeOf(resp.Reservations))
		fmt.Println("len(resp.Reservations)", len(resp.Reservations))
		fmt.Println("Type of r:", reflect.TypeOf(r))
		fmt.Println("r:", r)
		fmt.Println("len(resp.Reservations[r].Instances):", len(resp.Reservations[r].Instances))
		for _, inst := range resp.Reservations[r].Instances {

			if inst != nil {

				fmt.Println("inst:", reflect.TypeOf(inst))
				////fmt.Println("Instance Id: ", *inst.InstanceId)
				//fmt.Println("*inst.InstanceId:", *inst.InstanceId)
				//fmt.Println("inst.InstanceId:", reflect.TypeOf(*inst.InstanceId))
				////fmt.Println("len(inst):", len(inst))

				//instances = append(instances, *inst.InstanceId)
				/*
					if *inst.InstanceId != nil {
						fmt.Println("*inst.InstanceId had positive value")
						instances = append(instances, inst)
						instance := inst
					}
				*/
				//ti := taggedInstance{instanceId: *inst.InstanceId, instance: inst}
				////instances = append(instances, inst)
				//instances = append(instances, ti)

				//instance := inst
				fmt.Println("-----------------------------------------------------------------------------------------")
				fmt.Println("*inst.InstanceId:", reflect.TypeOf(*inst.InstanceId))
				fmt.Println(*inst.InstanceId)
				//fmt.Println(inst)
				fmt.Println("-----------------------------------------------------------------------------------------")
				//fmt.Println("type of instance:", reflect.TypeOf(instance))
				//fmt.Println("len(instances)", len(instances))

				//taggedInstances[*inst.InstanceId] = inst
				taggedInstances = append(taggedInstances, inst)
			}
		}
	}
	//instances := "test"
	//return instances
	return taggedInstances
}

// BlockDevice containes all fields necessary
// to take a snapshot
type BlockDevice struct {
	InstanceId string
	DeviceName string
	VolumeId   string
}

// Get block devices from instance
//func readBlockDeviceFromInstance(instance []taggedInstance) {
func readBlockDeviceFromInstance(instance *ec2.Instance) (map[string]interface{}, error) {

	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil

	//func readBlockDeviceFromInstance(instances map[string]*ec2.Instance) {

	//for id, instance := range instances {
	//	fmt.Println("Instance id:", id)
	//	fmt.Println("*ec2.Instance")
	//	fmt.Println("Instance id from instance:", *instance.InstanceId)

	//}

	instanceBlockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs != nil {
			instanceBlockDevices[*bd.Ebs.VolumeId] = bd
			fmt.Println("bd:", bd)
		}
	}

	if len(instanceBlockDevices) == 0 {
		return nil, nil
	}

	volIDs := make([]*string, 0, len(instanceBlockDevices))
	for volID := range instanceBlockDevices {
		volIDs = append(volIDs, aws.String(volID))
		fmt.Println("volID:", volID)
	}

	// Call DescribeVolumes to get vol size and
	volResp, err := svc.DescribeVolumes(&ec2.DescribeVolumesInput{
		VolumeIds: volIDs,
	})
	if err != nil {
		return nil, err
	}
	//fmt.Println("volResp:", volResp)

	for _, vol := range volResp.Volumes {
		instanceBd := instanceBlockDevices[*vol.VolumeId]
		bd := make(map[string]interface{})

		bd["volume_id"] = *vol.VolumeId

		if instanceBd.Ebs != nil && instanceBd.Ebs.DeleteOnTermination != nil {
			bd["delete_on_terminaiton"] = *instanceBd.Ebs.DeleteOnTermination
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
			//blockDevices["root"] = bd
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

	//return nil, nil
	return blockDevices, nil
	/*

		fmt.Println("instance", instance)

		bd := BlockDevice{
			//InstanceId: *instance.InstanceId,
			InstanceId: instance,
		}

		input := &ec2.DescribeInstanceAttributeInput{
			Attribute:  aws.String("blockDeviceMapping"),
			InstanceId: aws.String(instance),
			//InstanceId: aws.String(*instance.InstanceId),
		}

		result, err := svc.DescribeInstanceAttribute(input)
		fmt.Println("reault:", reflect.TypeOf(result))
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

		var instanceBlockDevices []*ec2.InstanceBlockDeviceMapping
		instanceBlockDevices = result.BlockDeviceMappings
		//instanceBlockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
		for i, bd := range instanceBlockDevices {
			//for i, bd := range instance.BlockDevicesMappings {
			if bd.Ebs != nil {
				fmt.Println("index:", i, "Block Device:", bd)
				//instanceBlockDevices[*bd.Ebs.VolumeId] = bd
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

	*/
} // !- readBlockDeviceFromInstance()

func blockDeviceIsRoot(bd *ec2.InstanceBlockDeviceMapping, instance *ec2.Instance) bool {
	return bd.DeviceName != nil &&
		instance.RootDeviceName != nil &&
		*bd.DeviceName == *instance.RootDeviceName
}

func main() {

	svc := service()
	_ = svc
	// instances is a pointer to a slice of string
	tag := "Test"
	instances := getTaggedInstances(tag)
	_ = instances

	// Needs to run on a single instance
	for _, i := range instances {
		fmt.Println("instance id:", *i.InstanceId)
		readBlockDeviceFromInstance(i)
	}

	//fmt.Println("Type:", reflect.TypeOf(instances))
	//fmt.Println("instances:", instances)

	/*
		for _, i := range instances {
			//	fmt.Println("Instance Id", i)

			//readBlockDeviceFromInstance(i)
			fmt.Println(i)
		}
	*/

	//readBlockDeviceFromInstance(instances)

}
