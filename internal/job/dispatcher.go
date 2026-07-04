package job

import (
	"log"

	"gorm.io/gorm"
)

type Dispatcher struct {
	db *gorm.DB
}

func NewDispatcher(db *gorm.DB) *Dispatcher {
	return &Dispatcher{db: db}
}

func (d *Dispatcher) CascadeSoftDeleteWorkspace(workspaceID string) {
	go func() {
		if err := cascadeSoftDeleteWorkspace(d.db, workspaceID); err != nil {
			log.Printf("[job] Cascade soft delete failed for workspace %s: %v", workspaceID, err)
		} else {
			log.Printf("[job] Cascade soft delete completed for workspace %s", workspaceID)
		}
	}()
}
