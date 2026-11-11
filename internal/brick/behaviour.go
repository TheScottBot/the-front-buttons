package brick

import (
	"context"
	"errors"
	"reflect"
)

// DefaultReasonCategory is shown on a key whose brick failed without naming why.
const DefaultReasonCategory = "failed"

// Pressable is a brick that does something when its key is pressed
// (specification section 2). A brick implements only the interfaces it uses.
type Pressable interface {
	Press(pressContext context.Context) error
}

// ReasonCategorised is an error that names its own short reason category, such
// as "network" or "unauthorised", for the key's error state. The category is
// shown and counted; the error's full text, which may hold a URL, is not.
type ReasonCategorised interface {
	error
	ReasonCategory() string
}

// ReasonCategoryOf returns an error's reason category, looking through wrapping,
// or DefaultReasonCategory if it names none.
func ReasonCategoryOf(failure error) string {
	if categorised, isCategorised := errors.AsType[ReasonCategorised](failure); isCategorised {
		return categorised.ReasonCategory()
	}

	return DefaultReasonCategory
}

// AppearanceOf reads the Appearance embedded in a kind's parameters, as the
// schema requires every kind to embed it. It reports false for anything that is
// not a pointer to such a struct.
func AppearanceOf(parameters any) (Appearance, bool) {
	value := reflect.ValueOf(parameters)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return Appearance{Label: "", Icon: "", Background: ""}, false
	}

	for structField := range value.Elem().Type().Fields() {
		if structField.Anonymous && structField.Type == reflect.TypeFor[Appearance]() {
			appearance, _ := reflect.TypeAssert[Appearance](value.Elem().FieldByIndex(structField.Index))

			return appearance, true
		}
	}

	return Appearance{Label: "", Icon: "", Background: ""}, false
}
