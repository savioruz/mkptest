package dto

import (
	"oil/internal/domains/user/model"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Email        string  `json:"email"                   validate:"required,email"`
	Password     string  `json:"password"                validate:"required,min=8"`
	Level        string  `json:"level"                   validate:"omitempty,oneof=1 2 3"`
	FullName     *string `json:"full_name,omitempty"`
	ProfileImage *string `json:"profile_image,omitempty"`
	IsVerified   *bool   `json:"is_verified,omitempty"`
}

func (r *CreateUserRequest) ToModel(username string, hashedPassword string) model.User {
	level := r.Level
	if level == "" {
		level = constant.RoleUser
	}

	isVerified := false
	if r.IsVerified != nil {
		isVerified = *r.IsVerified
	}

	return model.User{
		ID:           uuid.NewString(),
		Email:        r.Email,
		Password:     hashedPassword,
		Level:        level,
		FullName:     r.FullName,
		ProfileImage: r.ProfileImage,
		IsVerified:   isVerified,
		Active:       true,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  username,
			ModifiedBy: username,
		},
	}
}

type UserResponse struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	Level        string  `json:"level"`
	FullName     *string `json:"full_name,omitempty"`
	ProfileImage *string `json:"profile_image,omitempty"`
	IsVerified   bool    `json:"is_verified"`
	LastLogin    *string `json:"last_login,omitempty"`
	Active       bool    `json:"active"`
	gDto.Metadata
}

func (r *UserResponse) FromModel(model model.User) {
	r.ID = model.ID
	r.Email = model.Email
	r.Level = model.Level
	r.FullName = model.FullName
	r.ProfileImage = model.ProfileImage
	r.IsVerified = model.IsVerified
	r.LastLogin = model.LastLogin
	r.Active = model.Active
	r.Metadata.FromModel(model.Metadata)
}

type UpdateUserRequest struct {
	Level        *string `json:"level,omitempty"         validate:"omitempty,oneof=1 2 3"`
	FullName     *string `json:"full_name,omitempty"`
	ProfileImage *string `json:"profile_image,omitempty"`
	IsVerified   *bool   `json:"is_verified,omitempty"`
	Active       *bool   `json:"active,omitempty"`
}

type UpdateProfileRequest struct {
	FullName     *string `json:"full_name,omitempty"`
	ProfileImage *string `json:"profile_image,omitempty"`
}

type GetUsersResponse struct {
	Users     []UserResponse `json:"users"`
	TotalPage int            `json:"total_page"`
	TotalData int            `json:"total_data"`
}

func (r *GetUsersResponse) FromModels(models []model.User, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Users = make([]UserResponse, len(models))
	for i, mod := range models {
		r.Users[i].FromModel(mod)
	}
}
