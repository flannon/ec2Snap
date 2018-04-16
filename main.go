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

// Make the ec2 service connection using
// environment variables to get auth tokens
func ec2Service() *ec2.EC2 {

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
func describeSnapshotBlockDevice(instance *ec2.Instance) (map[string]interface{}, error) {

	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil

	instanceBlockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs != nil {
			instanceBlockDevices[*bd.Ebs.VolumeId] = bd
		}
	}

	if len(instanceBlockDevices) == 0 {
		return nil, nil
	}

	volIDs := make([]*string, 0, len(instanceBlockDevices))
	for volID := range instanceBlockDevices {
		volIDs = append(volIDs, aws.String(volID))
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

		bd["volumeId"] = *vol.VolumeId

		if instanceBd.Ebs != nil && instanceBd.Ebs.DeleteOnTermination != nil {
			bd["deleteOnTermination"] = *instanceBd.Ebs.DeleteOnTermination
		}
		if vol.Size != nil {
			bd["volumeSize"] = *vol.Size
		}
		if vol.VolumeType != nil {
			bd["volumeType"] = *vol.VolumeType
		}
		if vol.Iops != nil {
			bd["iops"] = *vol.Iops
		}
		if blockDeviceIsRoot(instanceBd, instance) {
			blockDevices["root"] = bd
		} else {
			if instanceBd.DeviceName != nil {
				bd["deviceName"] = *instanceBd.DeviceName
			}
			if vol.Encrypted != nil {
				bd["encrypted"] = *vol.Encrypted
			}
			if vol.SnapshotId != nil {
				bd["snapshotId"] = *vol.SnapshotId
			}
			blockDevices["ebs"] = append(blockDevices["ebs"].([]map[string]interface{}), bd)
			//fmt.Println("blockDevice[ebs]", blockDevices["ebs"])
			//fmt.Println("Type of blockDevices[ebs]", reflect.TypeOf(blockDevices["ebs"]))
		}
	}

	return blockDevices, nil

} //!- describeSnapshotBlockDevice()

//!+blockDeviceIsRoot()
func blockDeviceIsRoot(bd *ec2.InstanceBlockDeviceMapping, instance *ec2.Instance) bool {
	return bd.DeviceName != nil &&
		instance.RootDeviceName != nil &&
		*bd.DeviceName == *instance.RootDeviceName
} // !-blockDeviceIsRoot()

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

// instanceVolumeIdsByTag returns volume ids for all volumes
// attached to a tagged instance.
func instanceVolumeIdsByTag(bd map[string]interface{}, t string) []string { //!+
	var volumeIds []string
	for dev, m := range bd {
		// v is a slice of map[string]interface {}
		// so first we have to split the slice
		// then dig the map values out.
		//fmt.Println("============")
		if dev == "ebs" {

			switch reflect.TypeOf(m).Kind() {
			case reflect.Slice:
				s := reflect.ValueOf(m)
				for i := 0; i < s.Len(); i++ {
					mi := s.Index(i).Interface()

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
					fmt.Printf("result.DeviceName: %v\n", result.DeviceName)
					//fmt.Printf("result.Encrypted: %v\n", result.Encrypted)
					//fmt.Printf("result.SnapshotId: %v\n", result.SnapshotId)

					volumeIds = append(volumeIds, result.VolumeId)
				}
			}
		} //
	} // for bd !-
	return volumeIds
} // !- getSnapshotVolumes()

type SnapshotVolumeInfo struct {
	name string
	id   string
}

func describeSnapshotVolumes(id string, t string) []SnapshotVolumeInfo {
	var snapshotVolumes []SnapshotVolumeInfo

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
		//return volumeIds
		return snapshotVolumes
	}
	if result.Volumes != nil {
		var snapshotVolumeOutput *ec2.DescribeVolumesOutput
		if err := mapstructure.Decode(result, &snapshotVolumeOutput); err != nil {
			panic(err)
		}
		var snapVol []*ec2.Volume
		if err := mapstructure.Decode(snapshotVolumeOutput.Volumes, &snapVol); err != nil {
			panic(err)
		}

		for _, v := range snapVol {

			// Volume attributes
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

			var n string
			// Process all the tags and and select 'Name' and t (the tag value
			// passed into the function)
			// Properly tagged volumes have both a Name tag and a search tag
			// if search tag matches and no name tag is present log the volume id as
			// an exception
			for _, vTag := range v.Tags {
				// Get the Name tag from the volume
				if *vTag.Key == "Name" {
					//fmt.Printf("Name.Key: %v\n", *vTag.Key)
					//fmt.Printf("Name.Value: %v\n", *vTag.Value)
					//fmt.Println("Name.VolumeId:", *v.VolumeId)
					//volumeIds = append(volumeIds, *v.VolumeId)
					n = *vTag.Value
				}
				// get the search tag from the SnapshotVolumeInfo
				if *vTag.Key == t {
					//fmt.Printf("t.Key: %v\n", *vTag.Key)
					//fmt.Printf("t.Value: %v\n", *vTag.Value)
					//fmt.Println("v.VolumeId:", *v.VolumeId)

					// add name and volume id to the SnapshotVolume struct
					if n != "" {
						id := *v.VolumeId
						//fmt.Printf("n: %v\n", n)
						//fmt.Printf("id: %v\n", id)
						sv := SnapshotVolumeInfo{name: n, id: id}
						snapshotVolumes = append(snapshotVolumes, sv)

					} else {
						fmt.Printf("Log the exception: %v has no Name tag\n", id)
					}
				}
			}
		}
	}
	return snapshotVolumes
}

////
// TagMap and ToEc2Tags were shamelessly barroed from
// https://github.com/visualphoenix/aws-go/blob/499b55c618daa2e2691a990f80285839e408d37b/aws/tag.go
////
// TagMap is a map of AWS tag key/values
type TagMap map[string]string

// ToEc2Tags converts a TagMap to a slice of
func ToEc2Tags(m *TagMap) []*ec2.Tag {
	var result []*ec2.Tag
	for k, v := range *m {
		result = append(result, &ec2.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return result
}

// mkSnapshot() takes takes a SnapshotVolume and
// makes a snapshot of the volume id, applying the name
// as the Name tag value
func mkSnapshot(svc *ec2.EC2, v SnapshotVolumeInfo, d string, t []*ec2.Tag, dr bool) {
	//fmt.Printf("id: %v\n", id)

	//fmt.Printf("name: %v\n", v.name)
	//fmt.Printf("id: %v\n", v.id)
	//fmt.Printf("d: %v\n", d)

	s := &ec2.CreateSnapshotInput{
		Description: aws.String(d),
		VolumeId:    aws.String(v.id),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeSnapshot),
				//Tags: []*ec2.Tag{
				//	{
				//		Key:   aws.String("Name"),
				//		Value: aws.String(v.name),
				//	},
				//}, // !-Tags
				Tags: t,
			},
		},
		DryRun: aws.Bool(dr),
	}
	result, err := svc.CreateSnapshot(s)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println(result)
}

//!+main
func main() {

	fmt.Println("This is a test")
	searchTag := "Test"
	description := "Manufactured by MakeSnapshotWorker"
	dr := true
	stm := make(TagMap)

	svc := ec2Service()

	instances := getTaggedInstances(searchTag)

	for _, i := range instances {
		ibd, _ := describeSnapshotBlockDevice(i)

		vids := instanceVolumeIdsByTag(ibd, searchTag)
		for _, id := range vids {
			snapVols := describeSnapshotVolumes(id, searchTag)
			for _, v := range snapVols {
				stm["Name"] = v.name
				//fmt.Println("stm[Name]:", stm["Name"])
				t := ToEc2Tags(&stm)
				//fmt.Println("type of t", reflect.TypeOf(t))
				//fmt.Println("type of v:", reflect.TypeOf(v))
				mkSnapshot(svc, v, description, t, dr)
			}
		}
	} //!-for
} //!-main
