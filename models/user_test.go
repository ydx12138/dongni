package models

import (
	"reflect"
	"testing"
)

func TestUserPhoneIsNullable(t *testing.T) {
	field, ok := reflect.TypeOf(User{}).FieldByName("Phone")
	if !ok {
		t.Fatal("Phone field is missing")
	}
	if field.Type.Kind() != reflect.Ptr || field.Type.Elem().Kind() != reflect.String {
		t.Fatalf("Phone must be a nullable *string, got %s", field.Type)
	}
}
