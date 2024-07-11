package repositories_clover

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/ostafen/clover/v2"
	clover_d "github.com/ostafen/clover/v2/document"
	"gitlab.com/nunet/device-management-service/db/repositories"
)

func handleDBError(err error) error {
	if err != nil {
		switch err {
		case clover.ErrDocumentNotExist:
			return repositories.NotFoundError
		case clover.ErrDuplicateKey:
			return repositories.InvalidDataError
		case repositories.ErrParsingModel:
			return err
		default:
			return errors.Join(repositories.DatabaseError, err)
		}
	}
	return nil
}

func toCloverDoc[T repositories.ModelType](data T) *clover_d.Document {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return clover_d.NewDocument()
	}
	mappedData := make(map[string]interface{})
	err = json.Unmarshal(jsonBytes, &mappedData)
	if err != nil {
		return clover_d.NewDocument()
	}

	doc := clover_d.NewDocumentOf(mappedData)
	return doc
}

func toModel[T repositories.ModelType](doc *clover_d.Document, isEntityRepo bool) (T, error) {
	var model T
	err := doc.Unmarshal(&model)
	if err != nil {
		return model, err
	}

	if !isEntityRepo {
		// we shouldn't try to update IDs of entity repositories as they might not
		// even have an ID at all
		model, err = repositories.UpdateField(model, "ID", doc.ObjectId())
		if err != nil {
			return model, err
		}
	}

	return model, nil
}

func fieldJSONTag[T repositories.ModelType](field string) string {
	fieldName := field
	if field, ok := reflect.TypeOf(*new(T)).FieldByName(field); ok {
		if tag, ok := field.Tag.Lookup("json"); ok {
			fieldName = strings.Split(tag, ",")[0]
		}
	}
	return fieldName
}
