package service

import (
	"errors"
	"scala-bot-v2/entity"
	"scala-bot-v2/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/api/drive/v3"
	"sync"
	"time"
)

type SongService struct {
	songRepository   *repository.SongRepository
	voiceRepository  *repository.VoiceRepository
	bandRepository   *repository.BandRepository
	driveRepository  *drive.Service
	driveFileService *DriveFileService
}

func NewSongService(songRepository *repository.SongRepository, voiceRepository *repository.VoiceRepository, bandRepository *repository.BandRepository,
	driveClient *drive.Service, driveFileService *DriveFileService) *SongService {
	return &SongService{
		songRepository:   songRepository,
		voiceRepository:  voiceRepository,
		bandRepository:   bandRepository,
		driveRepository:  driveClient,
		driveFileService: driveFileService,
	}
}

func (s *SongService) FindAll() ([]*entity.Song, error) {
	return s.songRepository.FindAll()
}

func (s *SongService) FindManyLiked(userID int64) ([]*entity.Song, error) {
	return s.songRepository.FindManyLiked(userID)
}

func (s *SongService) FindManyByDriveFileIDs(IDs []string) ([]*entity.Song, error) {
	return s.songRepository.FindManyByDriveFileIDs(IDs)
}

func (s *SongService) FindManyExtraLiked(userID int64, pageNumber int) ([]*entity.SongExtra, error) {
	return s.songRepository.FindManyExtraByPageNumberLiked(userID, pageNumber)
}

func (s *SongService) FindOneByID(ID primitive.ObjectID) (*entity.Song, error) {
	return s.songRepository.FindOneByID(ID)
}

func (s *SongService) FindOneByDriveFileID(driveFileID string) (*entity.Song, error) {
	return s.songRepository.FindOneByDriveFileID(driveFileID)
}

func (s *SongService) FindOneByName(driveFileID string) (*entity.Song, error) {
	return s.songRepository.FindOneByName(driveFileID)
}

func (s *SongService) FindOrCreateOneByDriveFileID(driveFileID string) (*entity.Song, *drive.File, error) {
	var driveFile *drive.File
	err := errors.New("fake error")
	for err != nil {
		driveFile, err = s.driveRepository.Files.Get(driveFileID).Fields("id, name, modifiedTime, webViewLink, parents").Do()
	}

	song, err := s.songRepository.FindOneByDriveFileID(driveFileID)
	if err != nil {
		song = &entity.Song{
			DriveFileID: driveFile.Id,
		}

		for _, parentFolderID := range driveFile.Parents {
			band, err := s.bandRepository.FindOneByDriveFolderID(parentFolderID)
			if err == nil {
				song.BandID = band.ID
				break
			}
		}
	}

	if songHasOutdatedPDF(song, driveFile) ||
		song.PDF.Name == "" || song.PDF.Key == "" || song.PDF.BPM == "" || song.PDF.Time == "" || song.PDF.WebViewLink == "" {
		song.PDF.Name = driveFile.Name
		song.PDF.Key, song.PDF.BPM, song.PDF.Time = s.driveFileService.GetMetadata(driveFile.Id)
		song.PDF.TgFileID = ""
		song.PDF.ModifiedTime = driveFile.ModifiedTime
		song.PDF.WebViewLink = driveFile.WebViewLink
	}

	song, err = s.songRepository.UpdateOne(*song)
	return song, driveFile, err
}

func (s *SongService) FindOrCreateManyByDriveFileIDs(driveFileIDs []string) ([]*entity.Song, []*drive.File, error) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(driveFileIDs))
	songs := make([]*entity.Song, len(driveFileIDs))
	driveFiles := make([]*drive.File, len(driveFileIDs))
	var err error
	for i := range driveFileIDs {
		go func(i int) {
			defer waitGroup.Done()

			song, driveFile, _err := s.FindOrCreateOneByDriveFileID(driveFileIDs[i])
			if _err != nil {
				err = _err
			}
			songs[i] = song
			driveFiles[i] = driveFile
		}(i)
	}
	waitGroup.Wait()

	return songs, driveFiles, err

}

func (s *SongService) UpdateOne(song entity.Song) (*entity.Song, error) {
	return s.songRepository.UpdateOne(song)
}

func (s *SongService) DeleteOneByDriveFileID(driveFileID string) error {
	err := s.driveRepository.Files.Delete(driveFileID).Do()
	if err != nil {
		return err
	}

	err = s.songRepository.DeleteOneByDriveFileID(driveFileID)
	if err != nil {
		return err
	}

	return nil
}

func (s *SongService) Like(songID primitive.ObjectID, userID int64) error {
	return s.songRepository.Like(songID, userID)
}

func (s *SongService) Dislike(songID primitive.ObjectID, userID int64) error {
	return s.songRepository.Dislike(songID, userID)
}

func (s *SongService) FindAllExtraByPageNumberSortedByEventsNumber(bandID primitive.ObjectID, pageNumber int) ([]*entity.SongExtra, error) {
	return s.songRepository.FindAllExtraByPageNumberSortedByEventsNumber(bandID, pageNumber)
}

func (s *SongService) FindAllExtraByPageNumberSortedByLatestEventDate(bandID primitive.ObjectID, pageNumber int) ([]*entity.SongExtra, error) {
	return s.songRepository.FindAllExtraByPageNumberSortedByLatestEventDate(bandID, pageNumber)
}

func (s *SongService) FindManyExtraByTag(tag string, bandID primitive.ObjectID, pageNumber int) ([]*entity.SongExtra, error) {
	return s.songRepository.FindManyExtraByTag(tag, bandID, pageNumber)
}

func songHasOutdatedPDF(song *entity.Song, driveFile *drive.File) bool {
	if song.PDF.ModifiedTime == "" || driveFile == nil {
		return true
	}

	pdfModifiedTime, err := time.Parse(time.RFC3339, song.PDF.ModifiedTime)
	if err != nil {
		return true
	}

	driveFileModifiedTime, err := time.Parse(time.RFC3339, driveFile.ModifiedTime)
	if err != nil {
		return true
	}

	if driveFileModifiedTime.After(pdfModifiedTime) {
		return true
	}

	return false
}

func (s *SongService) GetTags() ([]string, error) {
	return s.songRepository.GetTags()
}

func (s *SongService) TagOrUntag(tag string, songID primitive.ObjectID) (*entity.Song, error) {
	return s.songRepository.TagOrUntag(tag, songID)
}
