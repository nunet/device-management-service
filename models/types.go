package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Model is a base model for all entities.
type Model struct {
	ID        string `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// GetID returns the ID of the entity.
func (m Model) GetID() string {
	return m.ID
}

// BeforeCreate sets the ID and CreatedAt fields before creating a new entity.
// This is a GORM hook and should not be called directly.
// We can move this to generic repository create methods.
func (m *Model) BeforeCreate(tx *gorm.DB) error {
	m.ID = uuid.NewString()
	m.CreatedAt = time.Now()
	return nil
}

// BeforeUpdate sets the UpdatedAt field before updating an entity.
// This is a GORM hook and should not be called directly.
// We can move this to generic repository create methods.
func (m *Model) BeforeUpdate(tx *gorm.DB) error {
	m.UpdatedAt = time.Now()
	return nil
}
