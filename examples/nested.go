package main

import (
	"encoding/json"
	"fmt"
)

type Component struct {
	Components []struct {
		Id  string `json: "id"`
		Url string `json: "url"`
	} `json: components"`
}

func main() {
	var c Component

	b := []byte(`{"components":[{"id":"google","url":"http://google.com/"},{"id":"integralist","url":"http://integralist.co.uk/"},{"id":"sloooow","url":"http://stevesouders.com/cuzillion/?c0=hj1hfff5_0_f&c1=hc1hfff2_0_f&t=1439190969678"}]}`)

	json.Unmarshal(b, &c)

	fmt.Printf("%+v", c.Components[1]) // {Id: google Url:http://google.com/}

}
