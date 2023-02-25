package generator

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Item represent a 'product' or a 'service'
type Item struct {
	Name              string    `json:"name,omitempty" validate:"required"`
	Description       string    `json:"description,omitempty"`
	PriceExclVAT      string    `json:"unit_cost,omitempty"`
	PriceInclVAT      string    `json:"quantity,omitempty"`
	PayedPriceInclVAT string    `json:"payed_price_incl_vat,omitempty"`
	PayedPriceExclVAT string    `json:"payed_price_excl_vat,omitempty"`
	Tax               *Tax      `json:"tax,omitempty"`
	Discount          *Discount `json:"discount,omitempty"`

	_unitCost decimal.Decimal
	_quantity decimal.Decimal
}

// Prepare convert strings to decimal
func (i *Item) Prepare() error {
	// Unit cost
	unitCost, err := decimal.NewFromString(i.PriceExclVAT)
	if err != nil {
		return err
	}
	i._unitCost = unitCost

	// PriceInclVAT
	quantity, err := decimal.NewFromString(i.PriceInclVAT)
	if err != nil {
		return err
	}
	i._quantity = quantity

	// Tax
	if i.Tax != nil {
		if err := i.Tax.Prepare(); err != nil {
			return err
		}
	}

	// Discount
	if i.Discount != nil {
		if err := i.Discount.Prepare(); err != nil {
			return err
		}
	}

	return nil
}

// TotalWithoutTaxAndWithoutDiscount returns the total without tax and without discount
func (i *Item) TotalWithoutTaxAndWithoutDiscount() decimal.Decimal {
	quantity, _ := decimal.NewFromString(i.PriceInclVAT)
	price, _ := decimal.NewFromString(i.PriceExclVAT)
	total := price.Mul(quantity)

	return total
}

// TotalWithoutTaxAndWithDiscount returns the total without tax and with discount
func (i *Item) TotalWithoutTaxAndWithDiscount() decimal.Decimal {
	total := i.TotalWithoutTaxAndWithoutDiscount()

	// Check discount
	if i.Discount != nil {
		dType, dNum := i.Discount.getDiscount()

		if dType == DiscountTypeAmount {
			total = total.Sub(dNum)
		} else {
			// Percent
			toSub := total.Mul(dNum.Div(decimal.NewFromFloat(100)))
			total = total.Sub(toSub)
		}
	}

	return total
}

// TotalWithTaxAndDiscount returns the total with tax and discount
func (i *Item) TotalWithTaxAndDiscount() decimal.Decimal {
	return i.TotalWithoutTaxAndWithDiscount().Add(i.TaxWithTotalDiscounted())
}

// TaxWithTotalDiscounted returns the tax with total discounted
func (i *Item) TaxWithTotalDiscounted() decimal.Decimal {
	result := decimal.NewFromFloat(0)

	if i.Tax == nil {
		return result
	}

	totalHT := i.TotalWithoutTaxAndWithDiscount()
	taxType, taxAmount := i.Tax.getTax()

	if taxType == TaxTypeAmount {
		result = taxAmount
	} else {
		divider := decimal.NewFromFloat(100)
		result = totalHT.Mul(taxAmount.Div(divider))
	}

	return result
}

// appendColTo document doc
func (i *Item) appendColTo(options *Options, doc *Document) {
	// Get base Y (top of line)
	baseY := doc.pdf.GetY()

	// Name
	doc.pdf.SetX(ItemColNameOffset)
	doc.pdf.MultiCell(
		ItemColHTPriceOffset-ItemColNameOffset,
		3,
		doc.encodeString(i.Name),
		"",
		"",
		false,
	)

	// Description
	if len(i.Description) > 0 {
		doc.pdf.SetX(ItemColNameOffset)
		doc.pdf.SetY(doc.pdf.GetY() + 1)

		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.MultiCell(
			ItemColHTPriceOffset-ItemColNameOffset,
			3,
			doc.encodeString(i.Description),
			"",
			"",
			false,
		)

		// Reset font
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
	}

	// Compute line height
	colHeight := doc.pdf.GetY() - baseY

	// PriceExclVAT
	doc.pdf.SetY(baseY)
	doc.pdf.SetX(ItemColHTPriceOffset)
	doc.pdf.CellFormat(
		ItemColPriceInclVATOffset-ItemColHTPriceOffset,
		colHeight,
		doc.encodeString(doc.ac.FormatMoneyDecimal(i._unitCost)),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// PriceInclVAT
	doc.pdf.SetX(ItemColPriceInclVATOffset)
	doc.pdf.CellFormat(
		ItemColTaxOffset-ItemColPriceInclVATOffset,
		colHeight,
		doc.encodeString(doc.ac.FormatMoneyDecimal(i._quantity)),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Discount
	doc.pdf.SetX(ItemColDiscountOffset)
	if i.Discount == nil || i.Discount.Amount == "0.00" {
		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight,
			doc.encodeString("--"),
			"0",
			0,
			"",
			false,
			0,
			"",
		)
	} else {
		// If discount
		var discountDesc string
		decimalAmount, err := decimal.NewFromString(i.Discount.Amount)
		if err != nil {
			panic(err)
		}
		discountDesc = fmt.Sprintf("- %s", doc.ac.FormatMoneyDecimal(decimalAmount))

		// discount title
		// lastY := doc.pdf.GetY()
		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight/2,
			doc.encodeString(discountDesc),
			"0",
			0,
			"",
			false,
			0,
			"",
		)
		// discount desc
		doc.pdf.SetXY(ItemColDiscountOffset, baseY+(colHeight/2))
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.CellFormat(
			ItemColTotalTTCOffset-ItemColDiscountOffset,
			colHeight/2,
			doc.encodeString(i.Discount.Percent),
			"0",
			0,
			"LT",
			false,
			0,
			"",
		)

		// reset font and y
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
		doc.pdf.SetY(baseY)
	}

	// Tax
	doc.pdf.SetX(ItemColTaxOffset)
	if i.Tax == nil {
		// If no tax
		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight,
			doc.encodeString("--"),
			"0",
			0,
			"",
			false,
			0,
			"",
		)
	} else {
		decimalAmount, err := decimal.NewFromString(i.Tax.Amount)
		if err != nil {
			panic(err)
		}
		taxTitle := fmt.Sprintf("%s", doc.ac.FormatMoneyDecimal(decimalAmount))
		taxDesc := fmt.Sprintf("%s %s", i.Tax.Percent, doc.encodeString("%"))

		// tax title
		// lastY := doc.pdf.GetY()
		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight/2,
			doc.encodeString(taxTitle),
			"0",
			0,
			"LB",
			false,
			0,
			"",
		)

		// tax desc
		doc.pdf.SetXY(ItemColTaxOffset, baseY+(colHeight/2))
		doc.pdf.SetFont(doc.Options.Font, "", SmallTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.GreyTextColor[0],
			doc.Options.GreyTextColor[1],
			doc.Options.GreyTextColor[2],
		)

		doc.pdf.CellFormat(
			ItemColDiscountOffset-ItemColTaxOffset,
			colHeight/2,
			doc.encodeString(taxDesc),
			"0",
			0,
			"LT",
			false,
			0,
			"",
		)

		// reset font and y
		doc.pdf.SetFont(doc.Options.Font, "", BaseTextFontSize)
		doc.pdf.SetTextColor(
			doc.Options.BaseTextColor[0],
			doc.Options.BaseTextColor[1],
			doc.Options.BaseTextColor[2],
		)
		doc.pdf.SetY(baseY)
	}

	decimalAmount, err := decimal.NewFromString(i.PayedPriceInclVAT)
	if err != nil {
		panic(err)
	}
	// TOTAL TTC
	doc.pdf.SetX(ItemColTotalTTCOffset)
	doc.pdf.CellFormat(
		190-ItemColTotalTTCOffset,
		colHeight,
		doc.encodeString(doc.ac.FormatMoneyDecimal(decimalAmount)),
		"0",
		0,
		"",
		false,
		0,
		"",
	)

	// Set Y for next line
	doc.pdf.SetY(baseY + colHeight)
}
