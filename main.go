// main.go

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Constants for maximum allowance limits
const (
	maxDonationAllowance = 100000.0

	defaultPersonalAllowance = 60000.0
	maxPersonalAllowance     = 100000.0

	defaultMaxKReceipt = 50000.0
	maxAdminKReceipt   = 100000.0

	minPersonalAllowance = 10000.0
	minKReceiptDeduction = 0.0
)

type CalculationRequest struct {
	TotalIncome float64 `json:"totalIncome"`
	WHT         float64 `json:"wht"`
	Allowances  []struct {
		AllowanceType string  `json:"allowanceType"`
		Amount        float64 `json:"amount"`
	} `json:"allowances"`
}

type TaxLevel struct {
	Level string  `json:"level"`
	Tax   float64 `json:"tax"`
}

type CalculationResponse struct {
	Tax      float64    `json:"tax"`
	TaxLevel []TaxLevel `json:"taxLevel"`
}

type RefundResponse struct {
	RefundTax float64 `json:"taxRefund"`
}

func calculateTaxHandler(c echo.Context) error {
	var request CalculationRequest
	err := c.Bind(&request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	// Adjust allowance amounts to use maximum if they exceed the limit
	for i, allowance := range request.Allowances {
		switch allowance.AllowanceType {
		case "donation":
			if allowance.Amount > maxDonationAllowance {
				request.Allowances[i].Amount = maxDonationAllowance
			}
		case "k-receipt":
			if allowance.Amount > defaultMaxKReceipt {
				request.Allowances[i].Amount = defaultMaxKReceipt
			}
		default:
			request.Allowances[i].Amount = 0
		}
	}
	// Calculate taxable income after deductions
	taxableIncome := request.TotalIncome
	for _, allowance := range request.Allowances {
		taxableIncome -= allowance.Amount

	}
	taxableIncome -= defaultPersonalAllowance

	// Define tax brackets
	taxBrackets := []struct {
		MinIncome float64
		MaxIncome float64
		Rate      float64
	}{
		{0, 150000, 0},
		{150001, 500000, 0.1},
		{500001, 1000000, 0.15},
		{1000001, 2000000, 0.20},
		{2000001, 1e12, 0.35}, // arbitrarily large value to represent infinity
	}

	// Calculate tax based on tax brackets
	var totalTax float64
	var taxLevels []TaxLevel
	for _, bracket := range taxBrackets {
		// Calculate tax for this bracket
		var bracketTax float64
		if taxableIncome >= bracket.MinIncome {
			if taxableIncome <= bracket.MaxIncome {
				bracketTax = (taxableIncome - bracket.MinIncome + 1) * bracket.Rate
			} else {
				bracketTax = (bracket.MaxIncome - bracket.MinIncome + 1) * bracket.Rate
			}
			totalTax += bracketTax
		}
		taxLevels = append(taxLevels, TaxLevel{
			Level: fmt.Sprintf("%.0f-%.0f", bracket.MinIncome, bracket.MaxIncome),
			Tax:   bracketTax,
		})
	}

	// Ensure tax is not negative
	if totalTax < 0 {
		totalTax = 0
	}

	response := CalculationResponse{
		Tax:      totalTax,
		TaxLevel: taxLevels,
	}

	// Apply withholding tax
	if totalTax < request.WHT {
		taxRefund := request.WHT - totalTax
		response := RefundResponse{
			RefundTax: taxRefund,
		}
		return c.JSON(http.StatusOK, response)

	}

	return c.JSON(http.StatusOK, response)
}
func main() {
	port := os.Getenv("PORT")
	//admin_user := os.Getenv("ADMIN_USER")
	//admin_pass := os.Getenv("ADMIN_PASS")
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.POST("/tax/calculations", calculateTaxHandler)

	// Start server

	e.Logger.Fatal(e.Start(":" + port))
}
