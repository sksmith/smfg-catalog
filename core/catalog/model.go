package catalog

// Product is a value object. A SKU able to be produced by the factory.
type Product struct {
	Sku  string `json:"sku"`
	Upc  string `json:"upc"`
	Name string `json:"name"`
}
