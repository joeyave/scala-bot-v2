package service

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"scala-bot-v2/entity"
	"scala-bot-v2/repository"
)

type MembershipService struct {
	membershipRepository *repository.MembershipRepository
}

func NewMembershipService(membershipRepository *repository.MembershipRepository) *MembershipService {
	return &MembershipService{
		membershipRepository: membershipRepository,
	}
}

func (s *MembershipService) FindAll() ([]*entity.Membership, error) {
	return s.membershipRepository.FindAll()
}

func (s *MembershipService) FindOneByID(ID primitive.ObjectID) (*entity.Membership, error) {
	return s.membershipRepository.FindOneByID(ID)
}
func (s *MembershipService) FindMultipleByEventID(ID primitive.ObjectID) ([]*entity.Membership, error) {
	return s.membershipRepository.FindMultipleByEventID(ID)
}

func (s *MembershipService) UpdateOne(membership entity.Membership) (*entity.Membership, error) {
	memberships, err := s.membershipRepository.FindMultipleByUserIDAndEventID(membership.UserID, membership.EventID)
	if err == nil {
		for _, mb := range memberships {
			if mb.RoleID == membership.RoleID {
				membership.ID = mb.ID
				break
			}
		}
	}

	return s.membershipRepository.UpdateOne(membership)
}

func (s *MembershipService) DeleteOneByID(ID primitive.ObjectID) error {
	return s.membershipRepository.DeleteOneByID(ID)
}
