package elements

type Element1 struct {
	ElementField1 int `json:"int"`
}

type Element2 map[string]*[]int
