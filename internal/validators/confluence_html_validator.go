package confluencevalidators

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = confluenceHtmlValidator{}

type confluenceHtmlValidator struct {
}

// Description describes the validation in plain text formatting.
func (validator confluenceHtmlValidator) Description(_ context.Context) string {
	return fmt.Sprintf("string is not valid confluence HTML.")
}

// MarkdownDescription describes the validation in Markdown formatting.
func (validator confluenceHtmlValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

// Validate performs the validation.
func (v confluenceHtmlValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()

	validHtmlError := isValidConfluenceHtmlInternal(value)

	if validHtmlError != nil {
		response.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			request.Path,
			"Invalid Confluence HTML Specified",
			validHtmlError.Error()))
	}
	return
}

func isValidConfluenceHtmlInternal(htmlStr string) error {
	r := strings.NewReader(htmlStr)
	d := xml.NewDecoder(r)

	// Configure the decoder for HTML; leave off strict and autoclose for XHTML
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity
	for {
		tt, err := d.Token()
		_ = tt
		switch err {
		case io.EOF:
			if strings.Contains(htmlStr, "<br>") || strings.Contains(htmlStr, "<br/>") {
				return fmt.Errorf("Invalid Html specified. Suggestion: %s", "Conflence requires: <br />")
			}

			if strings.Contains(htmlStr, "<hr>") || strings.Contains(htmlStr, "<hr/>") {
				return fmt.Errorf("Invalid Html specified. Suggestion: %s", "Conflence requires: <hr />")
			}
			return nil

		case nil:
		default:
			return err // Oops, something wasn't right
		}
	}
}

func IsValidConfluenceHtml() validator.String {
	return confluenceHtmlValidator{}
}
