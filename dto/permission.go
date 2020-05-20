package dto

type Permission struct {
	URL        string `json:"Url`
	IsWritable bool   `json:"IsWritable"`
}

type ClientPermission struct {
	Name string       `json:"Name"`
	Key  string       `json:"Key"`
	Perm []Permission `json:"Perm"`
}
