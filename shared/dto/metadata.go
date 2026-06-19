package dto

import (
	"oil/shared/constant"
	"oil/shared/model"
)

type Metadata struct {
	CreatedAt  string `json:"created_at"`
	ModifiedAt string `json:"modified_at"`
	CreatedBy  string `json:"created_by"`
	ModifiedBy string `json:"modified_by"`
}

func (m *Metadata) FromModel(model model.Metadata) {
	m.CreatedAt = model.CreatedAt.Format(constant.DateFormat)
	m.ModifiedAt = model.ModifiedAt.Format(constant.DateFormat)
	m.CreatedBy = model.CreatedBy
	m.ModifiedBy = model.ModifiedBy
}
