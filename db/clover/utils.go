// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package clover

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
			return repositories.ErrNotFound
		case clover.ErrDuplicateKey:
			return repositories.ErrInvalidData
		case repositories.ErrParsingModel:
			return err
		default:
			return errors.Join(repositories.ErrDatabase, err)
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
			if name := strings.Split(tag, ",")[0]; name != "" {
				fieldName = name
			}
		}
	}
	return fieldName
}
