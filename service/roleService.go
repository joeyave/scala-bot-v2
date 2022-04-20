package service

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"scala-bot-v2/entity"
	"scala-bot-v2/repository"
)

type RoleService struct {
	roleRepository *repository.RoleRepository
}

func NewRoleService(roleRepository *repository.RoleRepository) *RoleService {
	return &RoleService{
		roleRepository: roleRepository,
	}
}

func (s *RoleService) FindAll() ([]*entity.Role, error) {
	return s.roleRepository.FindAll()
}

func (s *RoleService) FindOneByID(ID primitive.ObjectID) (*entity.Role, error) {
	return s.roleRepository.FindOneByID(ID)
}

func (s *RoleService) UpdateOne(role entity.Role) (*entity.Role, error) {
	return s.roleRepository.UpdateOne(role)
}
