// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/oschwald/geoip2-golang"
	"gorm.io/gorm"
)

// BaseDBModel is a base model for all entities. It'll be mainly used for database
// records.
type BaseDBModel struct {
	ID        string `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// GetID returns the ID of the entity.
func (m BaseDBModel) GetID() string {
	return m.ID
}

// BeforeCreate sets the ID and CreatedAt fields before creating a new entity.
// This is a GORM hook and should not be called directly.
// We can move this to generic repository create methods.
func (m *BaseDBModel) BeforeCreate(_ *gorm.DB) error {
	m.ID = uuid.NewString()
	m.CreatedAt = time.Now()
	return nil
}

// BeforeUpdate sets the UpdatedAt field before updating an entity.
// This is a GORM hook and should not be called directly.
// We can move this to generic repository create methods.
func (m *BaseDBModel) BeforeUpdate(_ *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}

// GeoIPLocator returns the info about an IP.
type GeoIPLocator interface {
	Country(ipAddress net.IP) (*geoip2.Country, error)
	City(ipAddress net.IP) (*geoip2.City, error)
}
