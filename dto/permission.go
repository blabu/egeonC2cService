package dto

type Permission struct {
	URL        string `json:"url`
	IsWritable bool   `json:"isWritable"`
}

type ClientPermission struct {
	Key  string       `json:"key"`
	Perm []Permission `json:"perm"`
}
