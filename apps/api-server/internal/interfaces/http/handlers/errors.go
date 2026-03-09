package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
)

type apiErrorResponse struct {
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
	Details   any    `json:"details,omitempty"`
}

func respondError(c *gin.Context, status int, code, message string, details any) {
	payload := apiErrorResponse{
		Error:     message,
		ErrorCode: code,
		Details:   details,
	}
	c.JSON(status, payload)
}

func bindJSONOrAbort(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		handleBindJSONError(c, target, err)
		return false
	}
	return true
}

func handleBindJSONError(c *gin.Context, target any, err error) {
	var validationErrs validator.ValidationErrors
	switch {
	case errors.As(err, &validationErrs):
		requiredFields := make([]string, 0, len(validationErrs))
		invalidFields := make([]string, 0, len(validationErrs))
		for _, fieldErr := range validationErrs {
			fieldName := jsonFieldName(target, fieldErr.Field())
			if fieldName == "" {
				fieldName = strings.ToLower(fieldErr.Field())
			}
			if fieldErr.Tag() == "required" {
				requiredFields = appendUnique(requiredFields, fieldName)
				continue
			}
			invalidFields = appendUnique(invalidFields, fieldName)
		}

		switch {
		case len(requiredFields) > 0 && len(invalidFields) == 0:
			respondError(c, http.StatusBadRequest, "missing_required_fields", "Required fields are missing", gin.H{
				"fields": requiredFields,
			})
		case len(requiredFields)+len(invalidFields) > 0:
			fields := make([]string, 0, len(requiredFields)+len(invalidFields))
			fields = append(fields, requiredFields...)
			fields = append(fields, invalidFields...)
			respondError(c, http.StatusBadRequest, "invalid_request_fields", "Request fields are missing or invalid", gin.H{
				"fields": fields,
			})
		default:
			respondError(c, http.StatusBadRequest, "invalid_request", "Invalid request parameters", nil)
		}
	case isInvalidJSONError(err):
		respondError(c, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON", nil)
	default:
		respondError(c, http.StatusBadRequest, "invalid_request", "Invalid request parameters", nil)
	}
}

func isInvalidJSONError(err error) bool {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.As(err, &syntaxErr) ||
		errors.As(err, &typeErr)
}

func jsonFieldName(target any, fieldName string) string {
	targetType := reflect.TypeOf(target)
	if targetType == nil {
		return ""
	}
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}
	if targetType.Kind() != reflect.Struct {
		return ""
	}

	field, ok := targetType.FieldByName(fieldName)
	if !ok {
		return ""
	}

	tagValue := field.Tag.Get("json")
	if tagValue == "" {
		return ""
	}

	name := strings.Split(tagValue, ",")[0]
	if name == "" || name == "-" {
		return ""
	}

	return name
}

func appendUnique(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func requireQueryParams(c *gin.Context, names ...string) (map[string]string, bool) {
	values := make(map[string]string, len(names))
	missing := make([]string, 0, len(names))

	for _, name := range names {
		value := strings.TrimSpace(c.Query(name))
		if value == "" {
			missing = append(missing, name)
			continue
		}
		values[name] = value
	}

	if len(missing) > 0 {
		respondError(c, http.StatusBadRequest, "missing_required_fields", "Required query parameters are missing", gin.H{
			"fields": missing,
		})
		return nil, false
	}

	return values, true
}

func parseUintQueryParam(c *gin.Context, name string, rawValue string) (uint, bool) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(rawValue), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_query_parameter", name+" must be a valid unsigned integer", gin.H{
			"field": name,
		})
		return 0, false
	}

	return uint(parsed), true
}

func parseOptionalDateOrAbort(c *gin.Context, fieldName string, rawValue string) (time.Time, bool) {
	parsed, err := taskboardapp.ParseOptionalDate(rawValue, fieldName)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_date", err.Error(), gin.H{
			"field": fieldName,
		})
		return time.Time{}, false
	}

	return parsed, true
}
