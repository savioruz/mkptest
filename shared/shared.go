package shared

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"oil/config"
	"oil/shared/cache"
	"oil/shared/constant"
	"oil/shared/dto"
	"oil/shared/timezone"
	"reflect"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func ConvertStringToBool(value string) *bool {
	if value == "" {
		return nil
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert string to bool")

		return nil
	}

	return &boolValue
}

func CalculateTotalPage(total, limit int) (res int) {
	if total == 0 || limit <= 0 {
		res = 1
	} else {
		res = int(math.Ceil(float64(total) / float64(limit)))
	}

	return res
}

// TransformFields converts the fields of a struct into a map of updated fields.
func TransformFields(data interface{}, user string) map[string]any {
	val := reflect.ValueOf(data)
	typ := reflect.TypeOf(data)

	updatedFields := make(map[string]any)

	for index := range val.NumField() {
		field := val.Field(index)
		if field.IsZero() {
			continue
		}

		fieldName := typ.Field(index).Tag.Get("db")
		if fieldName == "" {
			continue
		}

		updatedFields[fieldName] = field.Interface()
	}

	updatedFields[constant.FieldModifiedAt] = timezone.Now()
	updatedFields[constant.FieldModifiedBy] = user

	return updatedFields
}

func FilterByID(id, fieldID, table string) dto.FilterGroup {
	return dto.FilterGroup{
		Filters: []any{
			dto.Filter{
				Field:    fieldID,
				Value:    id,
				Operator: dto.FilterOperatorEq,
				Table:    table,
			},
		},
	}
}

func BuildCacheKey(key string, postfix ...string) string {
	cfg := config.Get()
	parent := cfg.App.Name

	if len(postfix) > 0 {
		suffix := strings.Join(postfix, ":")

		return fmt.Sprintf("%s:cache:%s:%s", parent, key, suffix)
	}

	return fmt.Sprintf("%s:cache:%s", parent, key)
}

func BuildCacheKeyWithQuery(key string, queryParams dto.QueryParams, filter dto.FilterGroup) string {
	cfg := config.Get()
	parent := cfg.App.Name

	queryHash := generateQueryHash(queryParams, filter)

	return fmt.Sprintf("%s:cache:%s:%s", parent, key, queryHash)
}

func generateQueryHash(queryParams dto.QueryParams, filter dto.FilterGroup) string {
	queryData := struct {
		QueryParams dto.QueryParams `json:"query_params"`
		Filter      dto.FilterGroup `json:"filter"`
	}{
		QueryParams: queryParams,
		Filter:      filter,
	}

	jsonData, err := json.Marshal(queryData)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal query data for cache key")

		return fmt.Sprintf("page_%d_limit_%d_sortBy_%s_sortDir_%s",
			queryParams.Page, queryParams.Limit, queryParams.SortBy, queryParams.SortDir)
	}

	hash := md5.Sum(jsonData)

	return hex.EncodeToString(hash[:])
}

func InvalidateCaches(ctx context.Context, cache cache.RedisCache, key string) {
	if err := cache.Clear(ctx, BuildCacheKey(key, constant.Asterix)); err != nil {
		log.Error().Err(err).Msgf("failed to clear cache for key: %s", key)
	}
}

// GenerateUniqueFilename generates a unique filename with timestamp and original extension
func GenerateUniqueFilename(originalFilename string) string {
	timestamp := timezone.Now().Unix()
	parts := strings.Split(originalFilename, ".")
	extension := ""

	if len(parts) > 1 {
		extension = "." + parts[len(parts)-1]
	}

	hash := md5.Sum([]byte(fmt.Sprintf("%s_%d", originalFilename, timestamp)))
	hashStr := hex.EncodeToString(hash[:])

	return fmt.Sprintf("%d_%s%s", timestamp, hashStr[:8], extension)
}

type SearchField struct {
	Field   string
	Table   string
	DBField string
}

func SearchFieldsBuilder(request *http.Request, fields ...SearchField) (filters []any) {
	filters = []any{}

	for _, field := range fields {
		query := request.URL.Query().Get(field.Field)

		if query != "" {
			column := field.Field
			if field.DBField != "" {
				column = field.DBField
			}

			filters = append(filters, dto.Filter{
				Field:    column,
				Table:    field.Table,
				Value:    query,
				Operator: dto.FilterOperatorLike,
			})
		}
	}

	return filters
}
