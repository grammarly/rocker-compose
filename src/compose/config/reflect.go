package config

import (
	"reflect"
	"strings"
	"unicode"
)

var CompareSkipFields = []string{
	"Extends",
	"KillTimeout",
	"NetworkDisabled",
	"State",
	"KeepVolumes",

	// aliases
	"Command",
	"Link",
	"Label",
	"Hosts",
	"WorkingDir",
	"Environment",
}

func GetContainerFields() []string {
	fields := []string{}

	typeOfElem := reflect.ValueOf(&Container{}).Elem().Type()
	for i := 0; i < typeOfElem.NumField(); i++ {
		fields = append(fields, typeOfElem.Field(i).Name)
	}

	return fields
}

func GetComparableFields() []string {
	fields := []string{}

	for _, fieldName := range GetContainerFields() {
		// Skip some fields
		if unicode.IsLower((rune)(fieldName[0])) {
			continue
		}

		skip := false
		for _, f := range CompareSkipFields {
			if f == fieldName {
				skip = true
				break
			}
		}

		if !skip {
			fields = append(fields, fieldName)
		}
	}

	return fields
}

func GetYamlFields() []string {
	fields := []string{}

	for _, fieldName := range GetContainerFields() {
		field := GetYamlFieldName(fieldName)
		if field != "" && field != "-" {
			fields = append(fields, field)
		}
	}

	return fields
}

func GetYamlFieldName(fieldName string) string {
	field, _ := reflect.TypeOf(Container{}).FieldByName(fieldName)
	yamlTag := field.Tag.Get("yaml")
	split := strings.SplitN(yamlTag, ",", 2)
	return split[0]
}
