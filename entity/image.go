package entity

import "encoding/json"

type Image struct {
	Name       string
	BucketName string
}

func (i *Image) Marshal() ([]byte, error) {
	return json.Marshal(i)
}

func ImageFromJSON(data []byte) (*Image, error) {
	var u Image
	if err := json.Unmarshal(data, &u); err != nil {
		return nil, err
	}
	return &u, nil
}
