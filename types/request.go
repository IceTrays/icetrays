package types

type OpCpParams struct {
	Dir      string `json:"dir"`
	File     string `json:"file"`
	PinCount int    `json:"pin_count"`
	Crust    bool   `json:"crust"`
}
