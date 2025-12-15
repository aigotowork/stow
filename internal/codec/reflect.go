package codec

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

// FieldInfo contains information about a struct field that might need blob storage.
type FieldInfo struct {
	Name      string      // Field name
	Value     interface{} // Field value
	TagInfo   TagInfo     // Parsed tag info
	FieldType reflect.Type
}

// IsBlob determines if a value should be stored as a blob based on type, size, and tag.
//
// A value should be stored as a blob if:
//  1. Type is io.Reader
//  2. Type is []byte and size > threshold
//  3. Tag contains "file"
//  4. ForceFile option is set
func IsBlob(value interface{}, tagInfo TagInfo, threshold int64, forceFile bool) bool {
	// Check tag first
	if tagInfo.IsFile || forceFile {
		return true
	}

	// Check type
	val := reflect.ValueOf(value)
	if !val.IsValid() {
		return false
	}

	// io.Reader should always be stored as blob
	if _, ok := value.(io.Reader); ok {
		return true
	}

	// []byte larger than threshold should be stored as blob
	if val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8 {
		size := int64(val.Len())
		return size > threshold
	}

	return false
}

// ExtractBlobFields extracts all fields from a struct that should be stored as blobs.
func ExtractBlobFields(value interface{}, threshold int64) ([]FieldInfo, error) {
	val := reflect.ValueOf(value)

	// Dereference pointer
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("value is not a struct")
	}

	var blobFields []FieldInfo
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Parse tag
		tagStr := fieldType.Tag.Get("stow")
		tagInfo := ParseStowTag(tagStr)

		// Check if this field should be a blob
		fieldValue := field.Interface()
		if IsBlob(fieldValue, tagInfo, threshold, false) {
			blobFields = append(blobFields, FieldInfo{
				Name:      fieldType.Name,
				Value:     fieldValue,
				TagInfo:   tagInfo,
				FieldType: fieldType.Type,
			})
		}
	}

	return blobFields, nil
}

// ResolveNameField resolves the name_field reference in a struct.
// Returns the value of the referenced field as a string.
func ResolveNameField(structValue interface{}, nameField string) (string, error) {
	val := reflect.ValueOf(structValue)

	// Dereference pointer
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return "", fmt.Errorf("value is not a struct")
	}

	// Find the field
	field := val.FieldByName(nameField)
	if !field.IsValid() {
		return "", fmt.Errorf("field %s not found", nameField)
	}

	if !field.CanInterface() {
		return "", fmt.Errorf("field %s is not exported", nameField)
	}

	// Convert to string
	fieldValue := field.Interface()
	str, ok := fieldValue.(string)
	if !ok {
		return "", fmt.Errorf("field %s is not a string", nameField)
	}

	return str, nil
}

// ToMap converts a value to map[string]interface{}.
// This is used for serialization.
// Supports structs, maps, and scalar values (wrapped in a map).
func ToMap(value interface{}) (map[string]interface{}, error) {
	if value == nil {
		return map[string]interface{}{"$value": nil}, nil
	}

	val := reflect.ValueOf(value)

	// Dereference pointer
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return map[string]interface{}{"$value": nil}, nil
		}
		val = val.Elem()
	}

	// If already a map, return it
	if val.Kind() == reflect.Map {
		result := make(map[string]interface{})
		iter := val.MapRange()
		for iter.Next() {
			key := iter.Key().Interface()
			keyStr, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("map key is not a string")
			}
			result[keyStr] = iter.Value().Interface()
		}
		return result, nil
	}

	// Convert struct to map
	if val.Kind() != reflect.Struct {
		// Wrap scalar values in a map with "$value" key
		return map[string]interface{}{"$value": value}, nil
	}

	result := make(map[string]interface{})
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Get JSON tag name or use field name
		jsonTag := fieldType.Tag.Get("json")
		fieldName := fieldType.Name

		if jsonTag != "" && jsonTag != "-" {
			// Use JSON tag name
			parts := strings.Split(jsonTag, ",")
			fieldName = parts[0]
		}

		// Get field value
		fieldValue := field.Interface()

		// Handle nested structs recursively
		// Special case: time.Time should be treated as a value, not recursively converted
		if field.Kind() == reflect.Struct {
			// Check if it's time.Time
			if _, ok := fieldValue.(time.Time); ok {
				// time.Time should be treated as a value
				result[fieldName] = fieldValue
			} else {
				// Recursively convert nested struct to map
				nested, err := ToMap(fieldValue)
				if err != nil {
					return nil, fmt.Errorf("failed to convert nested struct field %s: %w", fieldName, err)
				}
				result[fieldName] = nested
			}
		} else if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			// Check if it's *time.Time
			if _, ok := fieldValue.(*time.Time); ok {
				// *time.Time should be treated as a value
				result[fieldName] = fieldValue
			} else {
				// Handle pointer to struct
				nested, err := ToMap(fieldValue)
				if err != nil {
					return nil, fmt.Errorf("failed to convert nested struct pointer field %s: %w", fieldName, err)
				}
				result[fieldName] = nested
			}
		} else {
			// Regular field
			result[fieldName] = fieldValue
		}
	}

	return result, nil
}

// FromMap converts map[string]interface{} to a target value.
// This is used for deserialization.
// Supports structs, maps, and scalar values (unwrapped from "$value" key).
func FromMap(data map[string]interface{}, target interface{}) error {
	val := reflect.ValueOf(target)

	// Must be a pointer
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	val = val.Elem()

	// Check if this is a wrapped scalar value
	if len(data) == 1 {
		if scalarValue, ok := data["$value"]; ok {
			// This is a wrapped scalar value
			if scalarValue == nil {
				val.Set(reflect.Zero(val.Type()))
				return nil
			}

			// If target is interface{}, just assign directly
			if val.Kind() == reflect.Interface {
				val.Set(reflect.ValueOf(scalarValue))
				return nil
			}

			// Try to set the value
			scalarVal := reflect.ValueOf(scalarValue)
			if scalarVal.Type().AssignableTo(val.Type()) {
				val.Set(scalarVal)
				return nil
			}
			if scalarVal.Type().ConvertibleTo(val.Type()) {
				val.Set(scalarVal.Convert(val.Type()))
				return nil
			}
			return fmt.Errorf("cannot convert %v to %v", scalarVal.Type(), val.Type())
		}
	}

	// If target is a map, just copy values
	if val.Kind() == reflect.Map {
		if val.IsNil() {
			val.Set(reflect.MakeMap(val.Type()))
		}

		for key, value := range data {
			val.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
		return nil
	}

	// Target must be a struct
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct, map, or scalar type")
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get field name from JSON tag or field name
		jsonTag := fieldType.Tag.Get("json")
		fieldName := fieldType.Name

		if jsonTag != "" && jsonTag != "-" {
			parts := strings.Split(jsonTag, ",")
			fieldName = parts[0]
		}

		// Get value from map
		mapValue, ok := data[fieldName]
		if !ok {
			continue
		}

		// Set field value
		if err := setFieldValue(field, mapValue); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldName, err)
		}
	}

	return nil
}

// setFieldValue sets a reflect.Value from an interface{}.
func setFieldValue(field reflect.Value, value interface{}) error {
	if value == nil {
		// Set to zero value
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	// Check if field is a struct and value is a map (nested struct case)
	if field.Kind() == reflect.Struct {
		if mapValue, ok := value.(map[string]interface{}); ok {
			// Recursively unmarshal nested struct
			fieldPtr := field.Addr().Interface()
			return FromMap(mapValue, fieldPtr)
		}
	}

	// Check if field is a pointer to struct
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
		if mapValue, ok := value.(map[string]interface{}); ok {
			// Create new struct instance if nil
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			// Recursively unmarshal nested struct
			return FromMap(mapValue, field.Interface())
		}
	}

	val := reflect.ValueOf(value)

	// Try direct assignment first
	if val.Type().AssignableTo(field.Type()) {
		field.Set(val)
		return nil
	}

	// Try conversion
	if val.Type().ConvertibleTo(field.Type()) {
		field.Set(val.Convert(field.Type()))
		return nil
	}

	return fmt.Errorf("cannot assign %v to %v", val.Type(), field.Type())
}

// IsSimpleType checks if a type is a simple (primitive) type.
func IsSimpleType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	}
	return false
}
