package gofakeit

import (
	"fmt"
	"math/rand"
	"strings"
)

type ProductInfo struct {
	Name        string   `json:"name" xml:"name"`
	Description string   `json:"description" xml:"description"`
	Categories  []string `json:"categories" xml:"categories"`
	Price       float64  `json:"price" xml:"price"`
	Features    []string `json:"features" xml:"features"`
	Color       string   `json:"color" xml:"color"`
	Material    string   `json:"material" xml:"material"`
	UPC         string   `json:"upc" xml:"upc"`
}

// Product will generate a random set of product information
func Product() *ProductInfo { return product(globalFaker.Rand) }

// Product will generate a random set of product information
func (f *Faker) Product() *ProductInfo { return product(f.Rand) }

func product(r *rand.Rand) *ProductInfo {
	// Categories
	categories := []string{}
	weightedCategory, _ := weighted(r, []any{1, 2, 3, 4}, []float32{1, 4, 3, 4})

	for i := 0; i < weightedCategory.(int); i++ {
		categories = append(categories, productCategory(r))
	}

	// Features
	features := []string{}
	for i := 0; i < number(r, 1, 5); i++ {
		features = append(features, productFeature(r))
	}

	product := &ProductInfo{
		Name:        productName(r),
		Description: productDescription(r),
		Categories:  categories,
		Price:       price(r, 3.00, 100.00),
		UPC:         productUPC(r),
		Features:    features,
		Color:       safeColor(r),
		Material:    productMaterial(r),
	}

	return product
}

// ProductName will generate a random product name
func ProductName() string { return productName(globalFaker.Rand) }

// ProductName will generate a random product name
func (f *Faker) ProductName() string { return productName(f.Rand) }

func productName(r *rand.Rand) string {
	name := getRandValue(r, []string{"product", "name"})
	switch number(r, 0, 9) {
	case 1:
		// Name + Adjective + Feature
		return title(fmt.Sprintf("%s %s %s", name, getRandValue(r, []string{"product", "adjective"}), productFeature(r)))
	case 2:
		// Adjective + Material + Name
		return title(fmt.Sprintf("%s %s %s", getRandValue(r, []string{"product", "adjective"}), productMaterial(r), name))
	case 3:
		// Color + Name + Suffix
		return title(fmt.Sprintf("%s %s %s", safeColor(r), name, getRandValue(r, []string{"product", "suffix"})))
	case 4:
		// Feature + Name + Adjective
		return title(fmt.Sprintf("%s %s %s", productFeature(r), name, getRandValue(r, []string{"product", "adjective"})))
	case 5:
		// Material + Color + Name
		return title(fmt.Sprintf("%s %s %s", productMaterial(r), safeColor(r), name))
	case 6:
		// Name + Suffix + Material
		return title(fmt.Sprintf("%s %s %s", name, getRandValue(r, []string{"product", "suffix"}), productMaterial(r)))
	case 7:
		// Adjective + Feature + Name
		return title(fmt.Sprintf("%s %s %s", getRandValue(r, []string{"product", "adjective"}), productFeature(r), name))
	case 8:
		// Color + Material + Name
		return title(fmt.Sprintf("%s %s %s", safeColor(r), productMaterial(r), name))
	case 9:
		// Suffix + Adjective + Name
		return title(fmt.Sprintf("%s %s %s", getRandValue(r, []string{"product", "suffix"}), getRandValue(r, []string{"product", "adjective"}), name))
	}

	// case: 0 - Adjective + Name + Suffix
	return title(fmt.Sprintf("%s %s %s", getRandValue(r, []string{"product", "adjective"}), name, getRandValue(r, []string{"product", "suffix"})))
}

// ProductDescription will generate a random product description
func ProductDescription() string { return productDescription(globalFaker.Rand) }

// ProductDescription will generate a random product description
func (f *Faker) ProductDescription() string { return productDescription(f.Rand) }

func productDescription(r *rand.Rand) string {
	desc := []string{}
	for i := 0; i < number(r, 1, 3); i++ {
		desc = append(desc, sentence(r, number(r, 5, 15)))
	}

	return strings.Join(desc, " ")
}

// ProductCategory will generate a random product category
func ProductCategory() string { return productCategory(globalFaker.Rand) }

// ProductCategory will generate a random product category
func (f *Faker) ProductCategory() string { return productCategory(f.Rand) }

func productCategory(r *rand.Rand) string {
	return getRandValue(r, []string{"product", "category"})
}

// ProductFeature will generate a random product feature
func ProductFeature() string { return productFeature(globalFaker.Rand) }

// ProductFeature will generate a random product feature
func (f *Faker) ProductFeature() string { return productFeature(f.Rand) }

func productFeature(r *rand.Rand) string {
	return getRandValue(r, []string{"product", "feature"})
}

// ProductMaterial will generate a random product material
func ProductMaterial() string { return productMaterial(globalFaker.Rand) }

// ProductMaterial will generate a random product material
func (f *Faker) ProductMaterial() string { return productMaterial(f.Rand) }

func productMaterial(r *rand.Rand) string {
	return getRandValue(r, []string{"product", "material"})
}

// ProductUPC will generate a random product UPC
func ProductUPC() string { return productUPC(globalFaker.Rand) }

// ProductUPC will generate a random product UPC
func (f *Faker) ProductUPC() string { return productUPC(f.Rand) }

func productUPC(r *rand.Rand) string {
	// The first digit of a UPC is a fixed digit (usually 0)
	upc := "0"

	// Generate the remaining 11 digits randomly
	for i := 1; i < 12; i++ {
		digit := number(r, 0, 9)
		upc += fmt.Sprintf("%d", digit)
	}

	return upc
}

func addProductLookup() {
	AddFuncLookup("product", Info{
		Display:     "Product",
		Category:    "product",
		Description: "An item created for sale or use",
		Example: `{
	"name": "olive copper monitor",
	"description": "Backwards caused quarterly without week it hungry thing someone him regularly. Whomever this revolt hence from his timing as quantity us these yours.",
	"categories": [
		"clothing",
		"tools and hardware"
	],
	"price": 7.61,
	"features": [
		"ultra-lightweight"
	],
	"color": "navy",
	"material": "brass",
	"upc": "012780949980"
}`,
		Output:      "map[string]any",
		ContentType: "application/json",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return product(r), nil
		},
	})

	AddFuncLookup("productname", Info{
		Display:     "Product Name",
		Category:    "product",
		Description: "Distinctive title or label assigned to a product for identification and marketing",
		Example:     "olive copper monitor",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return productName(r), nil
		},
	})

	AddFuncLookup("productdescription", Info{
		Display:     "Product Description",
		Category:    "product",
		Description: "Explanation detailing the features and characteristics of a product",
		Example:     "Backwards caused quarterly without week it hungry thing someone him regularly. Whomever this revolt hence from his timing as quantity us these yours.",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return productDescription(r), nil
		},
	})

	AddFuncLookup("productcategory", Info{
		Display:     "Product Category",
		Category:    "product",
		Description: "Classification grouping similar products based on shared characteristics or functions",
		Example:     "clothing",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return productCategory(r), nil
		},
	})

	AddFuncLookup("productfeature", Info{
		Display:     "Product Feature",
		Category:    "product",
		Description: "Specific characteristic of a product that distinguishes it from others products",
		Example:     "ultra-lightweight",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return productFeature(r), nil
		},
	})

	AddFuncLookup("productmaterial", Info{
		Display:     "Product Material",
		Category:    "product",
		Description: "The substance from which a product is made, influencing its appearance, durability, and properties",
		Example:     "brass",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return productMaterial(r), nil
		},
	})

	AddFuncLookup("productupc", Info{
		Display:     "Product UPC",
		Category:    "product",
		Description: "Standardized barcode used for product identification and tracking in retail and commerce",
		Example:     "012780949980",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return productUPC(r), nil
		},
	})
}
