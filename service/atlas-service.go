package service

import (
	"bpl/client"
	"bpl/repository"
)

type AtlasService struct {
	atlasRepository *repository.AtlasRepository
	latestAtlas     map[int]map[int]map[int][32]byte
}

func NewAtlasService() *AtlasService {
	return &AtlasService{
		atlasRepository: repository.NewAtlasRepository(),
		latestAtlas:     make(map[int]map[int]map[int][32]byte),
	}
}

func (a *AtlasService) initCache(userId int, eventId int) error {
	if a.latestAtlas[eventId] == nil {
		trees, err := a.atlasRepository.GetLatestTreesForEvent(eventId)
		if err != nil {
			return err
		}
		a.latestAtlas[eventId] = make(map[int]map[int][32]byte)
		for _, tree := range trees {
			if a.latestAtlas[eventId][tree.UserID] == nil {
				a.latestAtlas[eventId][tree.UserID] = map[int][32]byte{0: {}, 1: {}, 2: {}}
			}
			a.latestAtlas[eventId][tree.UserID][tree.Index] = tree.Nodes.GetHash()
		}
	}
	if a.latestAtlas[eventId][userId] == nil {
		a.latestAtlas[eventId][userId] = map[int][32]byte{0: {}, 1: {}, 2: {}}
	}
	return nil
}

func (a *AtlasService) SaveAtlasTrees(userId int, eventId int, trees []client.AtlasPassiveTree) error {
	if err := a.initCache(userId, eventId); err != nil {
		return err
	}
	hashValues := a.latestAtlas[eventId][userId]
	for index, tree := range trees {
		if repository.PassiveNodes(tree.Hashes).GetHash() != hashValues[index] {
			err := a.atlasRepository.CreateAtlasTree(userId, eventId, index, tree.Hashes)
			if err != nil {
				return err
			}
			hashValues[index] = repository.PassiveNodes(tree.Hashes).GetHash()
		}
	}
	return nil
}

func (a *AtlasService) GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*repository.AtlasTree, err error) {
	return a.atlasRepository.GetLatestAtlasesForEventAndTeam(eventId, teamId)
}

func (a *AtlasService) GetAtlasesForEventAndUser(userId int, eventId int) (atlas []*repository.AtlasTree, err error) {
	return a.atlasRepository.GetAtlasesForEventAndUser(eventId, userId)
}
