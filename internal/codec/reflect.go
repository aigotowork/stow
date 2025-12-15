package codec

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

const (
	// scalarValueKey is used to wrap scalar values in a map.
	scalarValueKey = "$value"
)

// dereferenceValue dereferences a pointer value if it's a pointer.
// Returns the dereferenced value or the original value if not a pointer.
func dereferenceValue(val reflect.Value) reflect.Value {
	if val.Kind() == reflect.Ptr {
		return val.Elem()
	}
	return val
}

// getFieldName extracts the field name from struct field, considering JSON tags.
// Returns the field name from the JSON tag if present, otherwise returns the struct field name.
func getFieldName(fieldType reflect.StructField) string {
	fieldName := fieldType.Name
	jsonTag := fieldType.Tag.Get("json")

	if jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		fieldName = parts[0]
	}

	return fieldName
}

// isTimeType checks if a value is time.Time or *time.Time.
func isTimeType(value interface{}) bool {
	switch value.(type) {
	case time.Time, *time.Time:
		return true
	default:
		return false
	}
}

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
	val := dereferenceValue(reflect.ValueOf(value))

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
	val := dereferenceValue(reflect.ValueOf(structValue))

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
		return map[string]interface{}{scalarValueKey: nil}, nil
	}

	val := reflect.ValueOf(value)

	// Dereference pointer
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return map[string]interface{}{scalarValueKey: nil}, nil
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
		// Wrap scalar values in a map with scalarValueKey key
		return map[string]interface{}{scalarValueKey: value}, nil
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

		fieldName := getFieldName(fieldType)
		fieldValue := field.Interface()

		// Handle nested structs recursively
		// Special case: time.Time should be treated as a value, not recursively converted
		if field.Kind() == reflect.Struct || (field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct) {
			if isTimeType(fieldValue) {
				// time.Time or *time.Time should be treated as a value
				result[fieldName] = fieldValue
			} else {
				// Recursively convert nested struct to map
				nested, err := ToMap(fieldValue)
				if err != nil {
					return nil, fmt.Errorf("failed to convert nested struct field %s: %w", fieldName, err)
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
// Supports structs, maps, and scalar values (unwrapped from scalarValueKey key).
func FromMap(data map[string]interface{}, target interface{}) error {
	val := reflect.ValueOf(target)

	// Must be a pointer
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	val = val.Elem()

	// Check if this is a wrapped scalar value
	if len(data) == 1 {
		if scalarValue, ok := data[scalarValueKey]; ok {
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

			// For complex types (slices, maps, structs), use setFieldValue
			// which has proper logic to handle type conversions
			if err := setFieldValue(val, scalarValue); err != nil {
				return fmt.Errorf("failed to set value: %w", err)
			}
			return nil
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

		fieldName := getFieldName(fieldType)

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
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	// Handle different field kinds
	switch field.Kind() {
	case reflect.Struct:
		return setStructField(field, value)

	case reflect.Ptr:
		return setPointerField(field, value)

	case reflect.Map:
		return setMapField(field, value)

	case reflect.Slice:
		return setSliceField(field, value)

	case reflect.Array:
		return setArrayField(field, value)

	default:
		return setScalarField(field, value)
	}
}

// setStructField handles struct field assignment.
func setStructField(field reflect.Value, value interface{}) error {
	if mapValue, ok := value.(map[string]interface{}); ok {
		return FromMap(mapValue, field.Addr().Interface())
	}
	return setScalarField(field, value)
}

// setPointerField handles pointer field assignment.
func setPointerField(field reflect.Value, value interface{}) error {
	if field.IsNil() {
		field.Set(reflect.New(field.Type().Elem()))
	}

	// If pointer to struct, handle as struct
	if field.Type().Elem().Kind() == reflect.Struct {
		if mapValue, ok := value.(map[string]interface{}); ok {
			return FromMap(mapValue, field.Interface())
		}
	}

	// For other pointer types, recursively set the pointed-to value
	return setFieldValue(field.Elem(), value)
}

// setMapField handles map field assignment.
func setMapField(field reflect.Value, value interface{}) error {
	mapValue, ok := value.(map[string]interface{})
	if !ok {
		return setScalarField(field, value)
	}

	if field.IsNil() {
		field.Set(reflect.MakeMap(field.Type()))
	}

	valueType := field.Type().Elem()
	for key, val := range mapValue {
		newVal := reflect.New(valueType).Elem()
		if err := setFieldValue(newVal, val); err != nil {
			return fmt.Errorf("failed to set map value for key %s: %w", key, err)
		}
		field.SetMapIndex(reflect.ValueOf(key), newVal)
	}
	return nil
}

// setSliceField handles slice field assignment.
func setSliceField(field reflect.Value, value interface{}) error {
	sliceValue, ok := value.([]interface{})
	if !ok {
		return setScalarField(field, value)
	}

	return setSequenceElements(field, sliceValue, func(length int) reflect.Value {
		return reflect.MakeSlice(field.Type(), length, length)
	})
}

// setArrayField handles array field assignment.
func setArrayField(field reflect.Value, value interface{}) error {
	sliceValue, ok := value.([]interface{})
	if !ok {
		return setScalarField(field, value)
	}

	if len(sliceValue) != field.Len() {
		return fmt.Errorf("array length mismatch: expected %d, got %d", field.Len(), len(sliceValue))
	}

	return setSequenceElements(field, sliceValue, func(length int) reflect.Value {
		return field // Arrays are already allocated
	})
}

// setSequenceElements sets elements for slice or array types.
// The makeContainer function creates the container (slice) or returns the existing one (array).
func setSequenceElements(field reflect.Value, values []interface{}, makeContainer func(int) reflect.Value) error {
	elemType := field.Type().Elem()
	container := makeContainer(len(values))

	for i, val := range values {
		newElem := reflect.New(elemType).Elem()
		if err := setFieldValue(newElem, val); err != nil {
			return fmt.Errorf("failed to set element at index %d: %w", i, err)
		}
		container.Index(i).Set(newElem)
	}

	// Only set the field if we created a new slice (not for arrays)
	if field.Kind() == reflect.Slice {
		field.Set(container)
	}

	return nil
}

// setScalarField handles scalar (primitive) field assignment with type conversion.
func setScalarField(field reflect.Value, value interface{}) error {
	val := reflect.ValueOf(value)

	if val.Type().AssignableTo(field.Type()) {
		field.Set(val)
		return nil
	}

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
