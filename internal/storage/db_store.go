package storage

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Link struct {
	ID          uint      `gorm:"primaryKey"`
	ShortCode   string    `gorm:"uniqueIndex;not null;type:varchar(15)"`
	OriginalURL string    `gorm:"uniqueIndex;not null"`
	CreatedAt   time.Time `gorm:"type:timestamp"`
}

func (l *Link) BeforeCreate(tx *gorm.DB) (err error) {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		return err
	}
	l.CreatedAt = time.Now().In(loc)
	return nil
}

type DbStore struct {
	db *gorm.DB
}

func NewDbStore(db *gorm.DB) (*DbStore, error) {
	err := db.AutoMigrate(&Link{})
	if err != nil {
		return nil, err
	}
	return &DbStore{db: db}, nil
}

func (s *DbStore) GetByCode(code string) (LinkData, error) {
	var link Link
	result := s.db.Where("short_code = ?", code).First(&link)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return LinkData{}, ErrNotFound
		}
		return LinkData{}, result.Error
	}
	return LinkData{
		URL:       link.OriginalURL,
		CreatedAt: link.CreatedAt,
	}, nil
}

func (s *DbStore) GetByURL(url string) (string, error) {
	var link Link
	result := s.db.Where("original_url = ?", url).First(&link)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		}
		return "", result.Error
	}
	return link.ShortCode, nil
}

func (s *DbStore) Save(url, code string) (string, error) {
	// First check if URL already exists
	existingCode, err := s.GetByURL(url)
	if err == nil {
		return existingCode, nil
	}
	if err != ErrNotFound {
		return "", err
	}

	link := Link{
		ShortCode:   code,
		OriginalURL: url,
	}

	result := s.db.Create(&link)
	if result.Error != nil {
		// To be safe, if Create fails we return ErrCodeCollision for code retry in handlers
		return "", ErrCodeCollision
	}

	return code, nil
}
