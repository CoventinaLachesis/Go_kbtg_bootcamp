package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestCalculateTaxHandler(t *testing.T) {
	// Define test cases
	tests := []struct {
		name     string
		request  CalculationRequest
		expected CalculationResponse
	}{
		{
			name: "Example 4",
			request: CalculationRequest{
				TotalIncome: 500000.0,
				WHT:         0.0,
				Allowances: []struct {
					AllowanceType string  `json:"allowanceType"`
					Amount        float64 `json:"amount"`
				}{
					{
						AllowanceType: "donation",
						Amount:        200000.0,
					},
				},
			},
			expected: CalculationResponse{
				Tax: 19000.0,
				TaxLevel: []TaxLevel{
					{Level: "0 - 150,000", Tax: 0.0},
					{Level: "150,001 - 500,000", Tax: 19000.0},
					{Level: "500,001 - 1,000,000", Tax: 0.0},
					{Level: "1,000,001 - 2,000,000", Tax: 0.0},
					{Level: "2,000,001 or more", Tax: 0.0},
				},
			},
		},
		{
			name: "Example 7",
			request: CalculationRequest{
				TotalIncome: 500000.0,
				WHT:         0.0,
				Allowances: []struct {
					AllowanceType string  `json:"allowanceType"`
					Amount        float64 `json:"amount"`
				}{
					{
						AllowanceType: "k-receipt",
						Amount:        200000.0,
					},
					{
						AllowanceType: "donation",
						Amount:        100000.0,
					},
				},
			},
			expected: CalculationResponse{
				Tax: 14000.0,
				TaxLevel: []TaxLevel{
					{Level: "0 - 150,000", Tax: 0.0},
					{Level: "150,001 - 500,000", Tax: 14000.0},
					{Level: "500,001 - 1,000,000", Tax: 0.0},
					{Level: "1,000,001 - 2,000,000", Tax: 0.0},
					{Level: "2,000,001 or more", Tax: 0.0},
				},
			},
		},
		// Add more test cases as needed
	}

	// Iterate over test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new echo instance
			e := echo.New()

			// Define a request body from the test case
			requestBody, err := json.Marshal(tc.request)
			assert.NoError(t, err)

			// Create a new HTTP request with the test case request body
			req := httptest.NewRequest(http.MethodPost, "/tax/calculations", strings.NewReader(string(requestBody)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Call the handler function
			err = calculateTaxHandler(c)

			// Assert that there is no error
			assert.NoError(t, err)

			// Assert the HTTP status code is OK
			assert.Equal(t, http.StatusOK, rec.Code)

			// Parse the response body into a CalculationResponse struct
			var response CalculationResponse
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Assert that the response matches the expected result
			assert.Equal(t, tc.expected.Tax, response.Tax)
			assert.Equal(t, tc.expected.TaxLevel, response.TaxLevel)
		})
	}
}
