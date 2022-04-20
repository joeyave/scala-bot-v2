package service

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"scala-bot-v2/entity"
	"scala-bot-v2/repository"
)

type BandService struct {
	bandRepository *repository.BandRepository
}

func NewBandService(bandRepository *repository.BandRepository) *BandService {
	return &BandService{
		bandRepository: bandRepository,
	}
}

func (s *BandService) FindAll() ([]*entity.Band, error) {
	return s.bandRepository.FindAll()
}

func (s *BandService) FindOneByID(ID primitive.ObjectID) (*entity.Band, error) {
	return s.bandRepository.FindOneByID(ID)
}

func (s *BandService) FindOneByDriveFolderID(driveFolderID string) (*entity.Band, error) {
	return s.bandRepository.FindOneByDriveFolderID(driveFolderID)
}

func (s *BandService) UpdateOne(band entity.Band) (*entity.Band, error) {
	return s.bandRepository.UpdateOne(band)
}
