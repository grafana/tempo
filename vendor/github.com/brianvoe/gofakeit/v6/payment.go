package gofakeit

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6/data"
)

// CurrencyInfo is a struct of currency information
type CurrencyInfo struct {
	Short string `json:"short" xml:"short"`
	Long  string `json:"long" xml:"long"`
}

// Currency will generate a struct with random currency information
func Currency() *CurrencyInfo { return currency(globalFaker.Rand) }

// Currency will generate a struct with random currency information
func (f *Faker) Currency() *CurrencyInfo { return currency(f.Rand) }

func currency(r *rand.Rand) *CurrencyInfo {
	index := r.Intn(len(data.Data["currency"]["short"]))
	return &CurrencyInfo{
		Short: data.Data["currency"]["short"][index],
		Long:  data.Data["currency"]["long"][index],
	}
}

// CurrencyShort will generate a random short currency value
func CurrencyShort() string { return currencyShort(globalFaker.Rand) }

// CurrencyShort will generate a random short currency value
func (f *Faker) CurrencyShort() string { return currencyShort(f.Rand) }

func currencyShort(r *rand.Rand) string { return getRandValue(r, []string{"currency", "short"}) }

// CurrencyLong will generate a random long currency name
func CurrencyLong() string { return currencyLong(globalFaker.Rand) }

// CurrencyLong will generate a random long currency name
func (f *Faker) CurrencyLong() string { return currencyLong(f.Rand) }

func currencyLong(r *rand.Rand) string { return getRandValue(r, []string{"currency", "long"}) }

// Price will take in a min and max value and return a formatted price
func Price(min, max float64) float64 { return price(globalFaker.Rand, min, max) }

// Price will take in a min and max value and return a formatted price
func (f *Faker) Price(min, max float64) float64 { return price(f.Rand, min, max) }

func price(r *rand.Rand, min, max float64) float64 {
	return math.Floor(float64Range(r, min, max)*100) / 100
}

// CreditCardInfo is a struct containing credit variables
type CreditCardInfo struct {
	Type   string `json:"type" xml:"type"`
	Number string `json:"number" xml:"number"`
	Exp    string `json:"exp" xml:"exp"`
	Cvv    string `json:"cvv" xml:"cvv"`
}

// CreditCard will generate a struct full of credit card information
func CreditCard() *CreditCardInfo { return creditCard(globalFaker.Rand) }

// CreditCard will generate a struct full of credit card information
func (f *Faker) CreditCard() *CreditCardInfo { return creditCard(f.Rand) }

func creditCard(r *rand.Rand) *CreditCardInfo {
	ccType := randomString(r, data.CreditCardTypes)
	return &CreditCardInfo{
		Type:   data.CreditCards[randomString(r, data.CreditCardTypes)].Display,
		Number: creditCardNumber(r, &CreditCardOptions{Types: []string{ccType}}),
		Exp:    creditCardExp(r),
		Cvv:    generate(r, strings.Repeat("#", int(data.CreditCards[randomString(r, data.CreditCardTypes)].Code.Size))),
	}
}

// CreditCardType will generate a random credit card type string
func CreditCardType() string { return creditCardType(globalFaker.Rand) }

// CreditCardType will generate a random credit card type string
func (f *Faker) CreditCardType() string { return creditCardType(f.Rand) }

func creditCardType(r *rand.Rand) string {
	return data.CreditCards[randomString(r, data.CreditCardTypes)].Display
}

// CreditCardOptions is the options for credit card number
type CreditCardOptions struct {
	Types []string `json:"types"`
	Bins  []string `json:"bins"` // optional parameter of prepended numbers
	Gaps  bool     `json:"gaps"`
}

// CreditCardNumber will generate a random luhn credit card number
func CreditCardNumber(cco *CreditCardOptions) string { return creditCardNumber(globalFaker.Rand, cco) }

// CreditCardNumber will generate a random luhn credit card number
func (f *Faker) CreditCardNumber(cco *CreditCardOptions) string { return creditCardNumber(f.Rand, cco) }

func creditCardNumber(r *rand.Rand, cco *CreditCardOptions) string {
	if cco == nil {
		cco = &CreditCardOptions{}
	}
	if cco.Types == nil || len(cco.Types) == 0 {
		cco.Types = data.CreditCardTypes
	}
	ccType := randomString(r, cco.Types)

	// Get Card info
	var cardInfo data.CreditCardInfo
	if info, ok := data.CreditCards[ccType]; ok {
		cardInfo = info
	} else {
		ccType = randomString(r, data.CreditCardTypes)
		cardInfo = data.CreditCards[ccType]
	}

	// Get length and pattern
	length := randomUint(r, cardInfo.Lengths)
	numStr := ""
	if len(cco.Bins) >= 1 {
		numStr = randomString(r, cco.Bins)
	} else {
		numStr = strconv.FormatUint(uint64(randomUint(r, cardInfo.Patterns)), 10)
	}
	numStr += strings.Repeat("#", int(length)-len(numStr))
	numStr = numerify(r, numStr)
	ui, _ := strconv.ParseUint(numStr, 10, 64)

	// Loop through until its a valid luhn
	for {
		valid := isLuhn(strconv.FormatUint(ui, 10))
		if valid {
			break
		}
		ui++
	}
	numStr = strconv.FormatUint(ui, 10)

	// Add gaps to number
	if cco.Gaps {
		for i, spot := range cardInfo.Gaps {
			numStr = numStr[:(int(spot)+i)] + " " + numStr[(int(spot)+i):]
		}
	}

	return numStr
}

// CreditCardExp will generate a random credit card expiration date string
// Exp date will always be a future date
func CreditCardExp() string { return creditCardExp(globalFaker.Rand) }

// CreditCardExp will generate a random credit card expiration date string
// Exp date will always be a future date
func (f *Faker) CreditCardExp() string { return creditCardExp(f.Rand) }

func creditCardExp(r *rand.Rand) string {
	month := strconv.Itoa(randIntRange(r, 1, 12))
	if len(month) == 1 {
		month = "0" + month
	}

	var currentYear = time.Now().Year() - 2000
	return month + "/" + strconv.Itoa(randIntRange(r, currentYear+1, currentYear+10))
}

// CreditCardCvv will generate a random CVV number
// Its a string because you could have 017 as an exp date
func CreditCardCvv() string { return creditCardCvv(globalFaker.Rand) }

// CreditCardCvv will generate a random CVV number
// Its a string because you could have 017 as an exp date
func (f *Faker) CreditCardCvv() string { return creditCardCvv(f.Rand) }

func creditCardCvv(r *rand.Rand) string { return numerify(r, "###") }

// isLuhn check is used for checking if credit card is a valid luhn card
func isLuhn(s string) bool {
	var t = [...]int{0, 2, 4, 6, 8, 1, 3, 5, 7, 9}
	odd := len(s) & 1
	var sum int
	for i, c := range s {
		if c < '0' || c > '9' {
			return false
		}
		if i&1 == odd {
			sum += t[c-'0']
		} else {
			sum += int(c - '0')
		}
	}
	return sum%10 == 0
}

// AchRouting will generate a 9 digit routing number
func AchRouting() string { return achRouting(globalFaker.Rand) }

// AchRouting will generate a 9 digit routing number
func (f *Faker) AchRouting() string { return achRouting(f.Rand) }

func achRouting(r *rand.Rand) string { return numerify(r, "#########") }

// AchAccount will generate a 12 digit account number
func AchAccount() string { return achAccount(globalFaker.Rand) }

// AchAccount will generate a 12 digit account number
func (f *Faker) AchAccount() string { return achAccount(f.Rand) }

func achAccount(r *rand.Rand) string { return numerify(r, "############") }

// BitcoinAddress will generate a random bitcoin address consisting of numbers, upper and lower characters
func BitcoinAddress() string { return bitcoinAddress(globalFaker.Rand) }

// BitcoinAddress will generate a random bitcoin address consisting of numbers, upper and lower characters
func (f *Faker) BitcoinAddress() string { return bitcoinAddress(f.Rand) }

func bitcoinAddress(r *rand.Rand) string {
	return randomString(r, []string{"1", "3"}) + password(r, true, true, true, false, false, number(r, 25, 34))
}

// BitcoinPrivateKey will generate a random bitcoin private key base58 consisting of numbers, upper and lower characters
func BitcoinPrivateKey() string { return bitcoinPrivateKey(globalFaker.Rand) }

// BitcoinPrivateKey will generate a random bitcoin private key base58 consisting of numbers, upper and lower characters
func (f *Faker) BitcoinPrivateKey() string { return bitcoinPrivateKey(f.Rand) }

func bitcoinPrivateKey(r *rand.Rand) string {
	var b strings.Builder
	for i := 0; i < 49; i++ {
		b.WriteString(randCharacter(r, base58))
	}
	return "5" + randomString(r, []string{"H", "J", "K"}) + b.String()
}

func addPaymentLookup() {
	AddFuncLookup("currency", Info{
		Display:     "Currency",
		Category:    "payment",
		Description: "Medium of exchange, often in the form of paper money or coins, used for trade and transactions",
		Example: `{
	"short": "IQD",
	"long": "Iraq Dinar"
}`,
		Output:      "map[string]string",
		ContentType: "application/json",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return currency(r), nil
		},
	})

	AddFuncLookup("currencyshort", Info{
		Display:     "Currency Short",
		Category:    "payment",
		Description: "Short 3-letter word used to represent a specific currency",
		Example:     "USD",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return currencyShort(r), nil
		},
	})

	AddFuncLookup("currencylong", Info{
		Display:     "Currency Long",
		Category:    "payment",
		Description: "Complete name of a specific currency used for official identification in financial transactions",
		Example:     "United States Dollar",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return currencyLong(r), nil
		},
	})

	AddFuncLookup("price", Info{
		Display:     "Price",
		Category:    "payment",
		Description: "The amount of money or value assigned to a product, service, or asset in a transaction",
		Example:     "92.26",
		Output:      "float64",
		Params: []Param{
			{Field: "min", Display: "Min", Type: "float", Default: "0", Description: "Minimum price value"},
			{Field: "max", Display: "Max", Type: "float", Default: "1000", Description: "Maximum price value"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			min, err := info.GetFloat64(m, "min")
			if err != nil {
				return nil, err
			}

			max, err := info.GetFloat64(m, "max")
			if err != nil {
				return nil, err
			}

			return price(r, min, max), nil
		},
	})

	AddFuncLookup("creditcard", Info{
		Display:     "Credit Card",
		Category:    "payment",
		Description: "Plastic card allowing users to make purchases on credit, with payment due at a later date",
		Example: `{
	"type": "UnionPay",
	"number": "4364599489953698",
	"exp": "02/24",
	"cvv": "300"
}`,
		Output:      "map[string]any",
		ContentType: "application/json",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return creditCard(r), nil
		},
	})

	AddFuncLookup("creditcardtype", Info{
		Display:     "Credit Card Type",
		Category:    "payment",
		Description: "Classification of credit cards based on the issuing company",
		Example:     "Visa",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return creditCardType(r), nil
		},
	})

	AddFuncLookup("creditcardnumber", Info{
		Display:     "Credit Card Number",
		Category:    "payment",
		Description: "Unique numerical identifier on a credit card used for making electronic payments and transactions",
		Example:     "4136459948995369",
		Output:      "string",
		Params: []Param{
			{
				Field: "types", Display: "Types", Type: "[]string", Default: "all",
				Options:     []string{"visa", "mastercard", "american-express", "diners-club", "discover", "jcb", "unionpay", "maestro", "elo", "hiper", "hipercard"},
				Description: "A select number of types you want to use when generating a credit card number",
			},
			{Field: "bins", Display: "Bins", Type: "[]string", Optional: true, Description: "Optional list of prepended bin numbers to pick from"},
			{Field: "gaps", Display: "Gaps", Type: "bool", Default: "false", Description: "Whether or not to have gaps in number"},
		},
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			types, err := info.GetStringArray(m, "types")
			if err != nil {
				return nil, err
			}
			if len(types) == 1 && types[0] == "all" {
				types = []string{}
			}

			bins, _ := info.GetStringArray(m, "bins")

			gaps, err := info.GetBool(m, "gaps")
			if err != nil {
				return nil, err
			}

			options := CreditCardOptions{
				Types: types,
				Gaps:  gaps,
			}

			if len(bins) >= 1 {
				options.Bins = bins
			}

			return creditCardNumber(r, &options), nil
		},
	})

	AddFuncLookup("creditcardexp", Info{
		Display:     "Credit Card Exp",
		Category:    "payment",
		Description: "Date when a credit card becomes invalid and cannot be used for transactions",
		Example:     "01/21",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return creditCardExp(r), nil
		},
	})

	AddFuncLookup("creditcardcvv", Info{
		Display:     "Credit Card CVV",
		Category:    "payment",
		Description: "Three or four-digit security code on a credit card used for online and remote transactions",
		Example:     "513",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return creditCardCvv(r), nil
		},
	})

	AddFuncLookup("achrouting", Info{
		Display:     "ACH Routing Number",
		Category:    "payment",
		Description: "Unique nine-digit code used in the U.S. for identifying the bank and processing electronic transactions",
		Example:     "513715684",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return achRouting(r), nil
		},
	})

	AddFuncLookup("achaccount", Info{
		Display:     "ACH Account Number",
		Category:    "payment",
		Description: "A bank account number used for Automated Clearing House transactions and electronic transfers",
		Example:     "491527954328",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return achAccount(r), nil
		},
	})

	AddFuncLookup("bitcoinaddress", Info{
		Display:     "Bitcoin Address",
		Category:    "payment",
		Description: "Cryptographic identifier used to receive, store, and send Bitcoin cryptocurrency in a peer-to-peer network",
		Example:     "1lWLbxojXq6BqWX7X60VkcDIvYA",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return bitcoinAddress(r), nil
		},
	})

	AddFuncLookup("bitcoinprivatekey", Info{
		Display:     "Bitcoin Private Key",
		Category:    "payment",
		Description: "Secret, secure code that allows the owner to access and control their Bitcoin holdings",
		Example:     "5vrbXTADWJ6sQBSYd6lLkG97jljNc0X9VPBvbVqsIH9lWOLcoqg",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return bitcoinPrivateKey(r), nil
		},
	})
}
