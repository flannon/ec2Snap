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
//func readBlockDeviceFromInstance(instance *ec2.Instance) {
func readBlockDeviceFromInstance(instance *ec2.Instance) (map[string]interface{}, error) {

	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil

	instanceBlockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs != nil {
			instanceBlockDevices[*bd.Ebs.VolumeId] = bd
			fmt.Println("bd:", bd)
			fmt.Println("Type of bd:", reflect.TypeOf(bd))
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
	fmt.Printf("Type of volResp: %v\n", reflect.TypeOf(volResp))
	if err != nil {
		return nil, err
	}

	for _, vol := range volResp.Volumes {
		instanceBd := instanceBlockDevices[*vol.VolumeId]
		fmt.Println("Type of instanceBd:", reflect.TypeOf(instanceBd))
		bd := make(map[string]interface{})

		fmt.Println("#############")
		// this is where the DescribeSnapshotVolume (bd) struct starts
		bd["volume_id"] = *vol.VolumeId
		fmt.Printf("bd[volume_id] %v\n", reflect.TypeOf(bd["volume_id"]))

		if instanceBd.Ebs != nil && instanceBd.Ebs.DeleteOnTermination != nil {
			bd["delete_on_termination"] = *instanceBd.Ebs.DeleteOnTermination
		}
		fmt.Printf("bd[delete_on_termination] %v\n", reflect.TypeOf(bd["delete_on_termination"]))
		if vol.Size != nil {
			bd["volume_size"] = *vol.Size
		}
		fmt.Printf("bd[volume_size] %v\n", reflect.TypeOf(bd["volume_size"]))
		if vol.VolumeType != nil {
			bd["volume_type"] = *vol.VolumeType
		}
		fmt.Printf("bd[volume_type] %v\n", reflect.TypeOf(bd["volume_type"]))
		if vol.Iops != nil {
			bd["iops"] = *vol.Iops
		}
		fmt.Printf("bd[iops] %v\n", reflect.TypeOf(bd["iops"]))
		if blockDeviceIsRoot(instanceBd, instance) {
			blockDevices["root"] = bd
			fmt.Printf("blockDevices[root] %v\n", reflect.TypeOf(blockDevices["root"]))
			fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")
		} else {

			if instanceBd.DeviceName != nil {
				bd["device_name"] = *instanceBd.DeviceName
			}
			fmt.Printf("bd[device_name] %v\n", reflect.TypeOf(bd["device_name"]))
			if vol.Encrypted != nil {
				bd["encrypted"] = *vol.Encrypted
			}
			fmt.Printf("bd[encrypted] %v\n", reflect.TypeOf(bd["encrypted"]))
			if vol.SnapshotId != nil {
				bd["snapshot_id"] = *vol.SnapshotId
			}
			fmt.Printf("bd[snapshot_id] %v\n", reflect.TypeOf(bd["snapshot_id"]))
			fmt.Println("#############")

			blockDevices["ebs"] = append(blockDevices["ebs"].([]map[string]interface{}), bd)
			fmt.Println("blockDevice[ebs]", blockDevices["ebs"])
			fmt.Println("Type of blockDevices[ebs]", reflect.TypeOf(blockDevices["ebs"]))
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

// InstanceBlockDevice struct
type InstanceBlockDevice struct {
	BlockDeviceMappings []*ec2.BlockDeviceMapping
}

// InstanceSnapshotBlockDevice struct
//type snapshotBlockDevice struct {
//	SnapshotId          string
//	VolumeId            string
//	DeleteOnTermination bool
//	VolumeSize          int64
//	VolumeType          string
//	Iops                int64
//	DeviceName          string
//	Encrypted           bool
//}

//type SnapshotBlockDevice struct {
//	volume_type           string
//	iops                  int64
//	device_name           string
//	encrypted             bool
//	snapshot_id           string
//	volume_id             string
//	delete_on_termination bool
//	volume_size           int64
//}
type SnapshotBlockDevice struct {
	Volume_size           int64
	Volume_type           string
	Iops                  int64
	Device_name           string
	Encrypted             bool
	Snapshot_id           string
	Volume_id             string
	Delete_on_termination bool
}

//type InstanceSnapshotBlockDevice struct {
//	snapshotBlockDevice `mapstructure:",squash"`
//	root                string
//}

type InstanceSnapshotBlockDevice struct {
	SnapshotBlockDevice `mapstructure:",squash"`
	Root                string
}

//!+main
func main() {

	//fmt.Println("Testy")
	svc := service()
	_ = svc
	// instances is a pointer to a slice of string
	tag := "Test"
	instances := getTaggedInstances(tag)
	//fmt.Println("Type of instances:", reflect.TypeOf(instances))

	for _, i := range instances {
		fmt.Println("Type of i:", reflect.TypeOf(i))
		fmt.Println("i.InstanceId:", *i.InstanceId)
		fmt.Println("-----------------------------------------------------------")

		//readBlockDeviceFromInstance(i)

		//ibd, _ := readBlockDeviceFromInstance(i)
		ibd, _ := readBlockDeviceFromInstance(i)
		_ = ibd

		/*
			fmt.Println("-----------------------------------------------------------")
			fmt.Printf("ibd: %v\n", ibd)
			fmt.Println("-----------------------------------------------------------")
			fmt.Printf("Type of ibd: %v\n", reflect.TypeOf(ibd))

			for k, v := range ibd {
				// v is a slice of map[string]interface {}
				// so first we have to split the slice
				// then dig the map values out.
				fmt.Println("============")
				if k == "ebs" {
					fmt.Printf("Key: %v, Value: %v\n", k, v)
					fmt.Printf("Type of value: %v\n", reflect.TypeOf(v))
					fmt.Printf("Type of value.kind: %v\n", reflect.TypeOf(v).Kind())
					fmt.Printf("Value of value: %v\n", reflect.ValueOf(v))

					switch reflect.TypeOf(v).Kind() {
					case reflect.Slice:
						s := reflect.ValueOf(v)

						for i := 0; i < s.Len(); i++ {
							fmt.Println(s.Index(i))
							fmt.Println("s.Index.(i).Kind()", s.Index(i).Kind())
							fmt.Println("s.Index.(i).Interface()", s.Index(i).Interface())
							mi := s.Index(i).Interface()
							fmt.Printf("mi: %+v\n", mi)
							fmt.Printf("Type of mi: %+v\n", reflect.TypeOf(mi))

							//var result Ec2ebs
							var result SnapshotBlockDevice

							if err := mapstructure.Decode(mi, &result); err != nil {
								panic(err)
							}
							fmt.Printf("%+v\n", result)
							fmt.Printf("result.Volume_size: %v\n", result.Volume_size)
							fmt.Printf("result.Volume_type: %v\n", result.Volume_type)
							fmt.Printf("result.Iops: %v\n", result.Iops)
							fmt.Printf("result.Device_name: %v\n", result.Device_name)
							fmt.Printf("result.Encrypted: %v\n", result.Encrypted)
							fmt.Printf("result.Snapshot_id: %v\n", result.Snapshot_id)
							fmt.Printf("result.Volume_id: %v\n", result.Volume_id)
							fmt.Printf("result.Delete_on_termination: %v\n", result.Delete_on_termination)
						}

					}
				}
			}
		*/
	} //!-for
} //!-main
