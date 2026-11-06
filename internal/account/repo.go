package account

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (ar *AccountRepository) CreateAccount(ctx context.Context, account *Account) error {
	if err := ar.db.WithContext(ctx).Create(account).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) Rename(ctx context.Context, id uint, newUsername string) error {
	result := ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("username", newUsername)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (ar *AccountRepository) RenameWithToken(ctx context.Context, id uint, newUsername string, token string) error {
	return ar.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Account{}).Where("id = ?", id).Update("username", newUsername)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Model(&Account{}).Where("id = ?", id).Update("token", token).Error; err != nil {
			return err
		}
		return nil
	})
}

func (ar *AccountRepository) ChangePassword(ctx context.Context, id uint, newPassword string) error {
	if err := ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("password_hash", newPassword).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) FindByID(ctx context.Context, id uint) (*Account, error) {
	var account Account
	if err := ar.db.WithContext(ctx).First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) FindByUsername(ctx context.Context, username string) (*Account, error) {
	var account Account
	if err := ar.db.WithContext(ctx).Where("username = ?", username).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (ar *AccountRepository) Login(ctx context.Context, id uint, token, refreshToken string) error {
	if err := ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Updates(map[string]interface{}{"token": token, "refresh_token": refreshToken, "refresh_expires_at": time.Now().Add(7 * 24 * time.Hour)}).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) Logout(ctx context.Context, id uint) error {
	if err := ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Updates(map[string]interface{}{"token": "", "refresh_token": "", "refresh_expires_at": nil}).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AccountRepository) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", accountID).Update("avatar_url", avatarURL).Error
}

func (ar *AccountRepository) UpdateToken(ctx context.Context, id uint, token string) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Update("token", token).Error
}

func (ar *AccountRepository) UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error {
	return ar.db.WithContext(ctx).Model(&Account{}).Where("id = ?", id).Updates(updates).Error
}

func (ar *AccountRepository) FindAll(ctx context.Context) ([]*Account, error) {
	var accounts []*Account
	if err := ar.db.WithContext(ctx).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (ar *AccountRepository) FindByRefreshToken(ctx context.Context, token string) (*Account, error) {
	var acc Account
	err := ar.db.WithContext(ctx).Where("refresh_token = ? AND refresh_expires_at > ?", token, time.Now()).First(&acc).Error
	return &acc, err
}
