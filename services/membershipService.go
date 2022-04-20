package services

import (
	"github.com/joeyave/scala-bot-v2/entities"
	"github.com/joeyave/scala-bot-v2/repositories"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MembershipService struct {
	membershipRepository *repositories.MembershipRepository
}

func NewMembershipService(membershipRepository *repositories.MembershipRepository) *MembershipService {
	return &MembershipService{
		membershipRepository: membershipRepository,
	}
}

func (s *MembershipService) FindAll() ([]*entities.Membership, error) {
	return s.membershipRepository.FindAll()
}

func (s *MembershipService) FindOneByID(ID primitive.ObjectID) (*entities.Membership, error) {
	return s.membershipRepository.FindOneByID(ID)
}
func (s *MembershipService) FindMultipleByEventID(ID primitive.ObjectID) ([]*entities.Membership, error) {
	return s.membershipRepository.FindMultipleByEventID(ID)
}

func (s *MembershipService) UpdateOne(membership entities.Membership) (*entities.Membership, error) {
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
