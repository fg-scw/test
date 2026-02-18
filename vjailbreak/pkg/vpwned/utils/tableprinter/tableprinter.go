package tableprinter

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

// PrintAsTable prints the fields and values of a slice of structs (or struct pointers) as a table.
func PrintAsTable(data interface{}, fields ...string) error {
	val := reflect.ValueOf(data)

	// Check if the input is a slice
	if val.Kind() != reflect.Slice {
		return fmt.Errorf("input must be a slice")
	}

	if val.Len() == 0 {
		fmt.Println("No data to display.")
		return nil
	}

	elementType := val.Type().Elem() // Get the type of elements in the slice
	isPtr := false

	// Check if the element type is a pointer
	if elementType.Kind() == reflect.Ptr {
		elementType = elementType.Elem() // Get the underlying type if it's a pointer
		isPtr = true
	}

	// Check if the element type is a struct
	if elementType.Kind() != reflect.Struct {
		return fmt.Errorf("slice elements must be structs or pointers to structs")
	}

	structType := elementType // The struct type (whether it was a pointer or not)

	// Determine which fields to include (either all or specified)
	var headers []string
	var fieldIndices []int
	if len(fields) == 0 {
		// If no fields are specified, include all
		numFields := structType.NumField()
		for i := 0; i < numFields; i++ {
			field := structType.Field(i)
			fieldName := field.Name

			// Skip protobuf-related fields
			if fieldName == "state" || fieldName == "sizeCache" || fieldName == "unknownFields" {
				continue
			}

			headers = append(headers, fieldName)
			fieldIndices = append(fieldIndices, i)
		}
	} else {
		// If fields are specified, only include those
		for _, fieldName := range fields {
			found := false
			numFields := structType.NumField()

			for i := 0; i < numFields; i++ {
				field := structType.Field(i)
				if field.Name == fieldName {
					headers = append(headers, fieldName)
					fieldIndices = append(fieldIndices, i)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("Warning: Field '%s' not found in struct\n", fieldName)
			}
		}
		if len(headers) == 0 {
			return fmt.Errorf("no valid fields specified")
		}
	}

	// Create a new tabwriter to format the output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Print Headers
	fmt.Fprintln(w, strings.Join(headers, "\t")+"\t")

	// Print Values
	for i := 0; i < val.Len(); i++ {
		element := val.Index(i)
		if isPtr {
			element = element.Elem() // Dereference the pointer if it's a slice of pointers
		}

		values := make([]string, len(headers))

		for k, j := range fieldIndices {
			fieldValue := element.Field(j)

			// If the field is unexported, try to call a getter method
			methodName := fmt.Sprintf("Get%s", strings.Title(structType.Field(j).Name))
			method := element.MethodByName(methodName)

			if method.IsValid() {
				// Call the getter method
				result := method.Call([]reflect.Value{})
				if len(result) > 0 {
					values[k] = fmt.Sprintf("%v", result[0].Interface())
				} else {
					values[k] = "<no value>" // Handle case where getter returns nothing
				}
			} else {
				values[k] = fmt.Sprintf("%v", fieldValue.Interface()) // Handle other types as before
			}
		}
		fmt.Fprintln(w, strings.Join(values, "\t")+"\t")
	}

	return w.Flush()
}
