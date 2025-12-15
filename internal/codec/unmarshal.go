package codec

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/aigotowork/stow/internal/blob"
)

// Unmarshaler handles deserialization from map[string]interface{} to target types.
// It detects blob references and loads them appropriately based on target type.
type Unmarshaler struct {
	blobManager *blob.Manager
	logger      Logger // Optional logger for warnings
}

// Logger interface for logging warnings (e.g., blob file not found).
type Logger interface {
	Warn(msg string, fields ...interface{})
}

// NewUnmarshaler creates a new unmarshaler.
func NewUnmarshaler(blobManager *blob.Manager) *Unmarshaler {
	return &Unmarshaler{
		blobManager: blobManager,
	}
}

// SetLogger sets a logger for warning messages.
func (u *Unmarshaler) SetLogger(logger Logger) {
	u.logger = logger
}

// Unmarshal unmarshals data into target, handling blob references.
//
// Blob handling:
//   - If target field is []byte: loads blob content into memory
//   - If target field is IFileData: returns file handle without loading content
//   - If blob file doesn't exist: logs warning and sets field to zero value
//
// Scalar value handling:
//   - If data contains only "$value" key, it's unwrapped as a scalar
func (u *Unmarshaler) Unmarshal(data map[string]interface{}, target interface{}) error {
	if target == nil {
		return fmt.Errorf("target is nil")
	}

	// Get reflect value of target
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	val = val.Elem()

	// Check if this is a wrapped scalar value
	if len(data) == 1 {
		if _, ok := data["$value"]; ok {
			// This is a wrapped scalar - use FromMap to unwrap
			return FromMap(data, target)
		}
	}

	// Handle interface{} target - just assign the map
	if val.Kind() == reflect.Interface {
		val.Set(reflect.ValueOf(data))
		return nil
	}

	// Handle map targets
	if val.Kind() == reflect.Map {
		return u.unmarshalToMap(data, val)
	}

	// Handle struct targets
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a struct or map")
	}

	return u.unmarshalToStruct(data, val)
}

// unmarshalToMap unmarshals data into a map.
func (u *Unmarshaler) unmarshalToMap(data map[string]interface{}, target reflect.Value) error {
	if target.IsNil() {
		target.Set(reflect.MakeMap(target.Type()))
	}

	for key, value := range data {
		// Check if value is a blob reference
		if m, ok := value.(map[string]interface{}); ok {
			if ref, isBlobRef := blob.FromMap(m); isBlobRef {
				// Load blob based on map value type (always []byte for maps)
				blobValue, err := u.loadBlobAsBytes(ref)
				if err != nil {
					u.logWarn(fmt.Sprintf("failed to load blob for key %s", key), err)
					continue
				}
				value = blobValue
			}
		}

		target.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}

	return nil
}

// unmarshalToStruct unmarshals data into a struct.
func (u *Unmarshaler) unmarshalToStruct(data map[string]interface{}, target reflect.Value) error {
	typ := target.Type()

	for i := 0; i < target.NumField(); i++ {
		field := target.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get field name from JSON tag
		fieldName := u.getFieldName(fieldType)

		// Get value from data
		value, ok := data[fieldName]
		if !ok {
			continue
		}

		// Check if value is a blob reference
		if m, ok := value.(map[string]interface{}); ok {
			if ref, isBlobRef := blob.FromMap(m); isBlobRef {
				// Load blob according to field type
				if err := u.loadBlobIntoField(ref, field); err != nil {
					u.logWarn(fmt.Sprintf("failed to load blob for field %s", fieldName), err)
					// Set to zero value
					field.Set(reflect.Zero(field.Type()))
				}
				continue
			}
		}

		// Regular field - set value
		if err := setFieldValue(field, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldName, err)
		}
	}

	return nil
}

// loadBlobIntoField loads a blob into a struct field based on field type.
func (u *Unmarshaler) loadBlobIntoField(ref *blob.Reference, field reflect.Value) error {
	// Check field type
	fieldType := field.Type()

	// If field is []byte, load content into memory
	if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Uint8 {
		data, err := u.loadBlobAsBytes(ref)
		if err != nil {
			return err
		}
		field.SetBytes(data)
		return nil
	}

	// If field is IFileData interface, return file handle
	// Check if field type implements IFileData
	// For now, we'll check if it's an interface and assume it's IFileData
	if fieldType.Kind() == reflect.Interface {
		fileData, err := u.blobManager.Load(ref)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(fileData))
		return nil
	}

	return fmt.Errorf("unsupported field type for blob: %v", fieldType)
}

// loadBlobAsBytes loads a blob's content as []byte.
func (u *Unmarshaler) loadBlobAsBytes(ref *blob.Reference) ([]byte, error) {
	return u.blobManager.LoadBytes(ref)
}

// loadBlobAsFileData loads a blob as a file handle (IFileData).
func (u *Unmarshaler) loadBlobAsFileData(ref *blob.Reference) (io.ReadCloser, error) {
	return u.blobManager.Load(ref)
}

// getFieldName gets the field name from JSON tag or uses field name.
func (u *Unmarshaler) getFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		return parts[0]
	}
	return field.Name
}

// logWarn logs a warning message if logger is set.
func (u *Unmarshaler) logWarn(msg string, err error) {
	if u.logger != nil {
		u.logger.Warn(msg, "error", err)
	}
}

// UnmarshalSimple unmarshals simple values (non-struct).
func (u *Unmarshaler) UnmarshalSimple(data interface{}, target interface{}) error {
	// Check if data is a blob reference
	if m, ok := data.(map[string]interface{}); ok {
		if ref, isBlobRef := blob.FromMap(m); isBlobRef {
			// Load blob as bytes
			bytes, err := u.loadBlobAsBytes(ref)
			if err != nil {
				return err
			}

			// Try to set target
			val := reflect.ValueOf(target)
			if val.Kind() != reflect.Ptr {
				return fmt.Errorf("target must be a pointer")
			}

			val = val.Elem()
			if val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8 {
				val.SetBytes(bytes)
				return nil
			}

			return fmt.Errorf("target type not compatible with blob data")
		}
	}

	// Regular data - just assign
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	val.Elem().Set(reflect.ValueOf(data))
	return nil
}
