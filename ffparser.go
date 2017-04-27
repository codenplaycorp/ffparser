package ffparser

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/helderfarias/ffparser/decorator"
	"github.com/helderfarias/ffparser/helper"
)

//FFParser flat file parser
type FFParser struct {
	decorators map[string]decorator.FieldDecorator
}

func NewSimpleParser() *FFParser {
	instance := FFParser{decorators: map[string]decorator.FieldDecorator{}}
	instance.decorators["Default"] = &decorator.DefaultDecorator{}
	instance.decorators["IntDecorator"] = &decorator.IntDecorator{}
	instance.decorators["Int64Decorator"] = &decorator.Int64Decorator{}
	return &instance
}

func (f *FFParser) ParseToText(src interface{}) (string, error) {
	mapFields, _ := Tags(src, "record")
	if len(mapFields) <= 0 {
		return "", fmt.Errorf("Could not fields public")
	}

	var buffer bytes.Buffer
	recordsField, _ := f.handlerRecordFieldsAndSort(mapFields)

	for _, record := range recordsField {
		decorator, err := f.factoryDecorator(src, record)
		if err != nil {
			return "", err
		}

		content, err := GetField(src, record.FieldName)
		if err != nil {
			return "", err
		}

		value, err := decorator.ToString(content)
		if err != nil {
			return "", err
		}

		if record.PaddingAlign == "right" {
			buffer.WriteString(helper.RightPadToLen(value, record.Delimiter, record.Size))
		} else if record.PaddingAlign == "left" {
			buffer.WriteString(helper.LeftPadToLen(value, record.Delimiter, record.Size))
		} else {
			return "", fmt.Errorf("Padding align invalid")
		}
	}

	return buffer.String(), nil
}

func (f *FFParser) CreateFromText(src interface{}, text string) error {
	mapFields, _ := Tags(src, "record")
	if len(mapFields) <= 0 {
		return fmt.Errorf("Could not fields public")
	}

	recordsField, _ := f.handlerRecordFieldsAndSort(mapFields)

	for _, record := range recordsField {
		decorator, err := f.factoryDecorator(src, record)
		if err != nil {
			return err
		}

		value, err := decorator.FromString(text[record.Start:record.End])
		if err != nil {
			return err
		}

		if err := SetField(src, record.FieldName, value); err != nil {
			return err
		}
	}

	return nil
}

func (f *FFParser) mapperTags(tagName string) map[string]string {
	tags := map[string]string{}

	for _, tagValue := range strings.Split(tagName, ",") {
		entry := strings.Split(tagValue, "=")

		if len(entry) >= 2 {
			tags[entry[0]] = entry[1]
		} else {
			tags[entry[0]] = ""
		}
	}

	return tags
}

func (f *FFParser) factoryDecorator(obj interface{}, record RecordField) (decorator.FieldDecorator, error) {
	if record.Decorator != "" {
		if decorator := f.decorators[record.Decorator]; decorator != nil {
			return decorator, nil
		}
	}

	typeField, err := GetFieldKind(obj, record.FieldName)
	if err != nil {
		return nil, err
	}

	switch typeField {
	case reflect.Int:
		return f.decorators["IntDecorator"], nil
	case reflect.Int64:
		return f.decorators["Int64Decorator"], nil
	default:
		return f.decorators["Default"], nil
	}
}

func (f *FFParser) handlerRecordFieldsAndSort(fields map[string]string) ([]RecordField, error) {
	records := []RecordField{}

	for fieldName, tagName := range fields {
		if tagName == "" {
			continue
		}

		tags := f.mapperTags(tagName)
		start := helper.ToInteger(tags["start"]) - 1
		end := helper.ToInteger(tags["end"])
		size := (end - start)
		decorator := tags["decorator"]
		delimiter := " "
		padAlign := "right"

		if tags["delimiter"] != "" {
			delimiter = tags["delimiter"]
		}

		if tags["padalign"] != "" {
			padAlign = tags["padalign"]
		}

		records = append(records, RecordField{
			FieldName:    fieldName,
			Start:        start,
			End:          end,
			Size:         size,
			Decorator:    decorator,
			Delimiter:    delimiter,
			PaddingAlign: padAlign,
		})
	}

	sort.Sort(RecordFieldSorted(records))

	return records, nil
}