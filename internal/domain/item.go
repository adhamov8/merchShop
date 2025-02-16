package domain

var MerchItems = map[string]int{
	"t-shirt":    80,
	"cup":        20,
	"book":       50,
	"pen":        10,
	"powerbank":  200,
	"hoody":      300,
	"umbrella":   200,
	"socks":      10,
	"wallet":     50,
	"pink-hoody": 500,
}

func IsValidMerchItem(name string) bool {
	_, exists := MerchItems[name]
	return exists
}

func GetItemPrice(name string) int {
	return MerchItems[name]
}
