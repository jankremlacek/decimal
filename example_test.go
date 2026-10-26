package decimal_test

import (
	"fmt"

	"github.com/jankremlacek/decimal"
)

func Example() {
	price := decimal.MustFromString("64500.4")
	qty := decimal.New(125, -2) // 1.25

	total := price.Mul(qty)
	fmt.Println(total)
	fmt.Println(total.StringFixed(2))

	share := total.Div(decimal.NewFromInt(3))
	fmt.Println(share)

	d, err := decimal.NewFromString("1.5e-8")
	if err != nil {
		panic(err)
	}
	fmt.Println(d, d.Cmp(decimal.Zero))
	// Output:
	// 80625.5
	// 80625.50
	// 26875.1666666666666667
	// 0.000000015 1
}
