// main.go

package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
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
	RefundTax string `json:"taxRefund"`
}

func formatLevel(min, max float64) string {
	if max == 1e12 {
		return fmt.Sprintf("%s or more", formatNumber(min))
	}
	return fmt.Sprintf("%s - %s", formatNumber(min), formatNumber(max))
}

// formatNumber formats a number with commas
func formatNumber(num float64) string {
	// Convert float64 to string
	str := strconv.FormatFloat(num, 'f', -1, 64)
	// Split integer part and decimal part
	parts := strings.Split(str, ".")
	integerPart := parts[0]
	// Add commas to integer part
	var formattedNumber string
	for i, c := range integerPart {
		if i > 0 && (len(integerPart)-i)%3 == 0 {
			formattedNumber += ","
		}
		formattedNumber += string(c)
	}
	// Add decimal part back if present
	if len(parts) > 1 {
		formattedNumber += "." + parts[1]
	}
	return formattedNumber
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
		level := formatLevel(bracket.MinIncome, bracket.MaxIncome)
		taxLevels = append(taxLevels, TaxLevel{
			Level: level,
			Tax:   bracketTax,
		})
	}

	// Ensure tax is not negative
	if totalTax < 0 {
		totalTax = 0
	}

	// Apply withholding tax
	if totalTax < request.WHT {
		taxRefund := request.WHT - totalTax
		response := RefundResponse{
			RefundTax: formatNumber(taxRefund),
		}
		return c.JSON(http.StatusOK, response)

	}

	totalTax -= request.WHT
	response := CalculationResponse{
		Tax:      totalTax,
		TaxLevel: taxLevels,
	}
	return c.JSON(http.StatusOK, response)
}
func main() {
	port := os.Getenv("PORT")
	db_url := os.Getenv("DATABASE_URL")

	//admin_user := os.Getenv("ADMIN_USER")
	//admin_pass := os.Getenv("ADMIN_PASS")

	//database
	db, err := sql.Open("postgres", db_url)
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	// Verify the database connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging the database:", err)
	}
	fmt.Println("Connected to the database")
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.POST("/tax/calculations", calculateTaxHandler)

	// Start server

	e.Logger.Fatal(e.Start(":" + port))
}
