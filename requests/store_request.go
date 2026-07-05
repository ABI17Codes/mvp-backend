package requests

type CreateStoreRequest struct {
	Name         string   `json:"storeName"`
	Description  string   `json:"description"`
	Number       string   `json:"number"`
	Logo         []string `json:"logo"`
	FacebookURL  string   `json:"facebookurl"`
	InstaURL     string   `json:"instaurl"`
	MapLink      string   `json:"maplink"`
	BannerImages []string `json:"bannerimages"`
	Slug         string   `json:"slug"`
	Address      string   `json:"address"`
}

type UpdateStoreRequest struct {
	Name          string   `json:"storeName"`
	Description   string   `json:"description"`
	Number        string   `json:"number"`
	Logo          []string `json:"logo"`
	FacebookURL   string   `json:"facebookurl"`
	InstaURL      string   `json:"instaurl"`
	MapLink       string   `json:"maplink"`
	BannerImages  []string `json:"bannerimages"`
	Slug          string   `json:"slug"`
	StoreTemplate string   `json:"store_template"`
}