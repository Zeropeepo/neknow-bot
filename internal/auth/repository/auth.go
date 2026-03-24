package repository

import(
	"context"
	"errors"
	"time"
	
	"github.com/google/uuid"
	"gorm.io/gorm"
	
	"github.com/Zeropeepo/neknow-bot/internal/auth/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/database/models"

)

type authRepository struct{
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) domain.Repository {
	return &authRepository{db: db}
}

// Mapper
func toModel(u *domain.User) *models.User {
	return &models.User{
		ID:				u.ID,
		Email:			u.Email,
		PasswordHash: 	u.PasswordHash,
		CreatedAt: 		u.CreatedAt,
		UpdatedAt: 		u.UpdatedAt,
	}
}

func toDomainUser(g *models.User) *domain.User{
	return &domain.User{
		ID:				g.ID,
		Email:			g.Email,
		PasswordHash:	g.PasswordHash,
		CreatedAt:		g.CreatedAt,
		UpdatedAt:		g.UpdatedAt,
	}
}

// Functions

func (r *authRepository) Create(ctx context.Context, user *domain.User) error {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	dbUser := toModel(user)
	return r.db.WithContext(ctx).Create(dbUser).Error
}

func (r *authRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error){
	var dbUser models.User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&dbUser)

	if errors.Is(result.Error, gorm.ErrRecordNotFound){
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return toDomainUser(&dbUser), nil
}

func (r *authRepository) FindByID(ctx context.Context, id string) (*domain.User, error){
	var dbUser models.User
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&dbUser)

	if errors.Is(result.Error, gorm.ErrRecordNotFound){
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return toDomainUser(&dbUser), nil
}