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
	"github.com/mitchellh/mapstructure"
)

// General refs
//   https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceAttribute
//   https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_instance.go
//	 https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeVolumes
//
// Snapshots and Tags
// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateSnapshot.html
// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/ec2-example-create-images.html
//

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
func getBlockDeviceFromInstance(instance *ec2.Instance) (map[string]interface{}, error) {

	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil

	instanceBlockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs != nil {
			instanceBlockDevices[*bd.Ebs.VolumeId] = bd
			//fmt.Println("bd:", bd)
			//fmt.Println("Type of bd:", reflect.TypeOf(bd))
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
	//fmt.Printf("volResp: %v\n", volResp)
	//fmt.Printf("Type of volResp: %v\n", reflect.TypeOf(volResp))
	if err != nil {
		return nil, err
	}

	for _, vol := range volResp.Volumes {
		instanceBd := instanceBlockDevices[*vol.VolumeId]
		//fmt.Println("Type of instanceBd:", reflect.TypeOf(instanceBd))
		bd := make(map[string]interface{})

		//fmt.Println("#############")
		// this is where the DescribeSnapshotVolume (bd) struct starts
		bd["volumeId"] = *vol.VolumeId
		//fmt.Printf("bd[volumeId] %v\n", reflect.TypeOf(bd["volumeId"]))

		if instanceBd.Ebs != nil && instanceBd.Ebs.DeleteOnTermination != nil {
			bd["deleteOnTermination"] = *instanceBd.Ebs.DeleteOnTermination
		}
		//fmt.Printf("bd[deleteOnTermination] %v\n", reflect.TypeOf(bd["deleteOnTermination"]))
		if vol.Size != nil {
			bd["volumeSize"] = *vol.Size
		}
		//fmt.Printf("bd[volumeSize] %v\n", reflect.TypeOf(bd["volumeSize"]))
		if vol.VolumeType != nil {
			bd["volumeType"] = *vol.VolumeType
		}
		//fmt.Printf("bd[volumeType] %v\n", reflect.TypeOf(bd["volumeType"]))
		if vol.Iops != nil {
			bd["iops"] = *vol.Iops
		}
		//fmt.Printf("bd[iops] %v\n", reflect.TypeOf(bd["iops"]))
		if blockDeviceIsRoot(instanceBd, instance) {
			blockDevices["root"] = bd
			//fmt.Printf("blockDevices[root] %v\n", reflect.TypeOf(blockDevices["root"]))
			//fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
		} else {

			if instanceBd.DeviceName != nil {
				bd["deviceName"] = *instanceBd.DeviceName
			}
			//fmt.Printf("bd[deviceName] %v\n", reflect.TypeOf(bd["deviceName"]))
			if vol.Encrypted != nil {
				bd["encrypted"] = *vol.Encrypted
			}
			//fmt.Printf("bd[encrypted] %v\n", reflect.TypeOf(bd["encrypted"]))
			if vol.SnapshotId != nil {
				bd["snapshotId"] = *vol.SnapshotId
			}
			//fmt.Printf("bd[snapshotId] %v\n", reflect.TypeOf(bd["snapshotId"]))
			//fmt.Println("#############")

			blockDevices["ebs"] = append(blockDevices["ebs"].([]map[string]interface{}), bd)
			//fmt.Println("blockDevice[ebs]", blockDevices["ebs"])
			//fmt.Println("Type of blockDevices[ebs]", reflect.TypeOf(blockDevices["ebs"]))
		}
	}

	return blockDevices, nil

} //!- getBlockDeviceFromInstance()

//!+blockDeviceIsRoot()
func blockDeviceIsRoot(bd *ec2.InstanceBlockDeviceMapping, instance *ec2.Instance) bool {
	return bd.DeviceName != nil &&
		instance.RootDeviceName != nil &&
		*bd.DeviceName == *instance.RootDeviceName
} // !-blockDeviceIsRoot()

// InstanceBlockDevice struct
//type InstanceBlockDevice struct {
//	BlockDeviceMappings []*ec2.BlockDeviceMapping
//}

// SnapshotBlockDevice contains all block device attributes
type SnapshotBlockDevice struct {
	VolumeId            string
	DeleteOnTermination bool
	VolumeSize          int64
	VolumeType          string
	Iops                int64
	DeviceName          string
	Encrypted           bool
	SnapshotId          string
}

func getInstanceVolumes(bd map[string]interface{}, t string) []string { //!+
	volIds := make([]string, 1)
	for k, v := range bd {
		// v is a slice of map[string]interface {}
		// so first we have to split the slice
		// then dig the map values out.
		fmt.Println("============")
		if k == "ebs" {
			//fmt.Printf("Key: %v, Value: %v\n", k, v)
			//fmt.Printf("Type of value: %v\n", reflect.TypeOf(v))
			//fmt.Printf("Type of value.kind: %v\n", reflect.TypeOf(v).Kind())
			//fmt.Printf("Value of value: %v\n", reflect.ValueOf(v))

			switch reflect.TypeOf(v).Kind() {
			case reflect.Slice:
				s := reflect.ValueOf(v)

				for i := 0; i < s.Len(); i++ {
					//fmt.Println(s.Index(i))
					//fmt.Println("s.Index.(i).Kind()", s.Index(i).Kind())
					//fmt.Println("s.Index.(i).Interface()", s.Index(i).Interface())
					mi := s.Index(i).Interface()
					//fmt.Printf("mi: %+v\n", mi)
					//fmt.Printf("Type of mi: %+v\n", reflect.TypeOf(mi))

					var result SnapshotBlockDevice

					if err := mapstructure.Decode(mi, &result); err != nil {
						panic(err)
					}
					//fmt.Printf("%+v\n", result)
					////fmt.Printf("VolumeId: %v\n", result.VolumeId)
					//fmt.Printf("result.DeleteOnTermination: %v\n", result.DeleteOnTermination)
					//fmt.Printf("result.VolumeSize: %v\n", result.VolumeSize)
					//fmt.Printf("result.VolumeType: %v\n", result.VolumeType)
					//fmt.Printf("result.Iops: %v\n", result.Iops)
					//fmt.Printf("result.DeviceName: %v\n", result.DeviceName)
					//fmt.Printf("result.Encrypted: %v\n", result.Encrypted)
					//fmt.Printf("result.SnapshotId: %v\n", result.SnapshotId)

					volIds = append(volIds, result.VolumeId)
					//describeInstanceVolume(result.VolumeId, t)
				}
			}
		} //
	} // for bd !-
	return volIds
} // !- getSnapshotVolumes()

type SnapshotVolume struct {
	volumeId    string
	snapshotId  string
	snapshotTag string
}

func describeSnapshotVolumes(id string, t string) []string {
	snapshotIds := make([]string, 2)

	//fmt.Println("id:", id)
	input := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("volume-id"),
				Values: []*string{
					aws.String(id),
				},
			},
			{
				Name: aws.String("tag:" + t),
				Values: []*string{
					aws.String("daily"), aws.String("weekly"), aws.String("monthly")},
			},
		},
	}

	result, err := svc.DescribeVolumes(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to aws.Error to get the Code
			// and Message from an error.
			fmt.Println(err.Error())
		}
		return snapshotIds
	}
	if result.Volumes != nil {
		//fmt.Printf("Type of result %v\n", reflect.TypeOf(result))
		////fmt.Println(result.Volumes)
		//fmt.Println("id:", id)

		var snapshotVolumeOutput *ec2.DescribeVolumesOutput
		if err := mapstructure.Decode(result, &snapshotVolumeOutput); err != nil {
			panic(err)
		}
		//fmt.Printf("sv %v\n", sv)
		//fmt.Printf("snapshotVolumeOutput.Volumes %v\n", snapshotVolumeOutput.Volumes)
		//fmt.Printf("type of snapshotVolumeOutput.Volumes %v\n", reflect.TypeOf(snapshotVolumeOutput.Volumes))

		var snapVol []*ec2.Volume
		if err := mapstructure.Decode(snapshotVolumeOutput.Volumes, &snapVol); err != nil {
			//panic(err)
			fmt.Println(err)
		}
		//fmt.Printf("sv: %v\n", snapVol)
		//fmt.Printf("type of sv: %v\n", reflect.TypeOf(snapVol))
		for _, v := range snapVol {
			//fmt.Println("v:", v)
			//fmt.Println("v.Attachments:", v.Attachments)
			//fmt.Println("type of v.Attachments:", reflect.TypeOf(v.Attachments))

			//fmt.Println("v.AvailabilityZone:", *v.AvailabilityZone)
			//fmt.Println("v.CreateTime:", *v.CreateTime)
			//fmt.Println("v.Encrypted:", *v.Encrypted)
			//fmt.Println("v.Iops:", *v.Iops)
			//fmt.Println("v.Size:", *v.Size)
			//fmt.Println("v.SnapshotId:", *v.SnapshotId)
			//fmt.Println("v.State:", *v.State)
			//fmt.Println("v.Tags:", v.Tags)
			//fmt.Println("type of v.Tags:", reflect.TypeOf(v.Tags))
			//fmt.Println("len(v.Tags):", len(v.Tags))
			//fmt.Println("v.Tags[1]:", v.Tags[0])
			// Process all the tags and find the one we're looking for
			for _, vTag := range v.Tags {
				//fmt.Printf("t: %v\n", vTag)
				if *vTag.Key == t {
					fmt.Println("We have a winner")
					fmt.Printf("t.Key: %v\n", *vTag.Key)
					fmt.Printf("t.Value: %v\n", *vTag.Value)
					fmt.Println("v.VolumeId:", *v.VolumeId)
					snapshotIds = append(snapshotIds, *v.VolumeId)
				}
			}

			//fmt.Println("v.VolumeType:", *v.VolumeType)
			//fmt.Println("type of v:", reflect.TypeOf(v))
		}
	}
	return snapshotIds
}

//!+main
func main() {

	service()
	instanceTagKey := "Test"
	//snapshotTagName := "say_hello_to_my_little_snapshot"
	instances := getTaggedInstances(instanceTagKey)

	for _, i := range instances {
		ibd, _ := getBlockDeviceFromInstance(i)

		//vids := getSnapshotVolumes(ibd, instanceTagKey)
		vids := getInstanceVolumes(ibd, instanceTagKey)
		for _, j := range vids {
			// describeSnapshotVolumes is where the heavy lifting happens
			// it needs to retrun all the information needed to snapshot each volume
			//sids := describeSnapshotVolumes(j, instanceTagKey)
			describeSnapshotVolumes(j, instanceTagKey)
		}

	} //!-for
} //!-main
