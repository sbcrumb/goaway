package profile

import (
	"context"
	"goaway/backend/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	GetAllSources(ctx context.Context) ([]database.Source, error)
	GetAllProfiles(ctx context.Context) ([]database.Profile, error)
	GetProfileByID(ctx context.Context, id uint) (*database.Profile, error)
	GetDefaultProfile(ctx context.Context) (*database.Profile, error)
	CreateProfile(ctx context.Context, p *database.Profile) error
	UpdateProfileName(ctx context.Context, id uint, name string) error
	DeleteProfile(ctx context.Context, id uint) error

	GetProfileSources(ctx context.Context, profileID uint) ([]database.ProfileSource, error)
	UpsertProfileSource(ctx context.Context, ps *database.ProfileSource) error
	SyncProfileSources(ctx context.Context, profileID uint, sources []database.Source) error

	GetActiveSourceDomains(ctx context.Context, profileID uint) ([]string, error)
	GetProfileCustomBlacklist(ctx context.Context, profileID uint, page, pageSize int, search string) ([]string, int64, error)
	AddProfileCustomBlacklist(ctx context.Context, profileID uint, domains []string) error
	RemoveProfileCustomBlacklist(ctx context.Context, profileID uint, domain string) error

	GetProfileWhitelist(ctx context.Context, profileID uint) ([]string, error)
	AddProfileWhitelist(ctx context.Context, profileID uint, domain string) error
	RemoveProfileWhitelist(ctx context.Context, profileID uint, domain string) error

	GetAllSubnets(ctx context.Context) ([]SubnetRule, error)
	CreateSubnet(ctx context.Context, s *database.SubnetProfile) error
	UpdateSubnet(ctx context.Context, id uint, cidr string, profileID uint) error
	DeleteSubnet(ctx context.Context, id uint) error

	SetClientProfile(ctx context.Context, ip string, profileID *uint) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetAllSources(ctx context.Context) ([]database.Source, error) {
	var sources []database.Source
	return sources, r.db.WithContext(ctx).Find(&sources).Error
}

func (r *repository) GetAllProfiles(ctx context.Context) ([]database.Profile, error) {
	var profiles []database.Profile
	return profiles, r.db.WithContext(ctx).Order("is_default DESC, name ASC").Find(&profiles).Error
}

func (r *repository) GetProfileByID(ctx context.Context, id uint) (*database.Profile, error) {
	var p database.Profile
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) GetDefaultProfile(ctx context.Context) (*database.Profile, error) {
	var p database.Profile
	if err := r.db.WithContext(ctx).Where("is_default = ?", true).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) CreateProfile(ctx context.Context, p *database.Profile) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *repository) UpdateProfileName(ctx context.Context, id uint, name string) error {
	return r.db.WithContext(ctx).Model(&database.Profile{}).Where("id = ? AND is_default = ?", id, false).Update("name", name).Error
}

func (r *repository) DeleteProfile(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Where("id = ? AND is_default = ?", id, false).Delete(&database.Profile{}).Error
}

func (r *repository) GetProfileSources(ctx context.Context, profileID uint) ([]database.ProfileSource, error) {
	var rows []database.ProfileSource
	return rows, r.db.WithContext(ctx).Where("profile_id = ?", profileID).Find(&rows).Error
}

func (r *repository) UpsertProfileSource(ctx context.Context, ps *database.ProfileSource) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "profile_id"}, {Name: "source_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"active"}),
	}).Create(ps).Error
}

// SyncProfileSources ensures every source has a row in profile_sources for this profile (active=true by default).
func (r *repository) SyncProfileSources(ctx context.Context, profileID uint, sources []database.Source) error {
	for _, s := range sources {
		ps := &database.ProfileSource{ProfileID: profileID, SourceID: s.ID, Active: true}
		if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(ps).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *repository) GetActiveSourceDomains(ctx context.Context, profileID uint) ([]string, error) {
	var domains []string
	err := r.db.WithContext(ctx).
		Table("blacklists b").
		Select("b.domain").
		Joins("INNER JOIN profile_sources ps ON b.source_id = ps.source_id").
		Where("ps.profile_id = ? AND ps.active = ?", profileID, true).
		Pluck("b.domain", &domains).Error
	return domains, err
}

func (r *repository) GetProfileCustomBlacklist(ctx context.Context, profileID uint, page, pageSize int, search string) ([]string, int64, error) {
	query := r.db.WithContext(ctx).Model(&database.ProfileCustomBlacklist{}).Where("profile_id = ?", profileID)
	if search != "" {
		query = query.Where("domain LIKE ?", "%"+search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var domains []string
	err := query.Offset((page - 1) * pageSize).Limit(pageSize).Pluck("domain", &domains).Error
	return domains, total, err
}

func (r *repository) AddProfileCustomBlacklist(ctx context.Context, profileID uint, domains []string) error {
	entries := make([]database.ProfileCustomBlacklist, 0, len(domains))
	for _, d := range domains {
		entries = append(entries, database.ProfileCustomBlacklist{ProfileID: profileID, Domain: d})
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&entries).Error
}

func (r *repository) RemoveProfileCustomBlacklist(ctx context.Context, profileID uint, domain string) error {
	return r.db.WithContext(ctx).Where("profile_id = ? AND domain = ?", profileID, domain).Delete(&database.ProfileCustomBlacklist{}).Error
}

func (r *repository) GetProfileWhitelist(ctx context.Context, profileID uint) ([]string, error) {
	var domains []string
	return domains, r.db.WithContext(ctx).Model(&database.ProfileWhitelist{}).Where("profile_id = ?", profileID).Pluck("domain", &domains).Error
}

func (r *repository) AddProfileWhitelist(ctx context.Context, profileID uint, domain string) error {
	entry := &database.ProfileWhitelist{ProfileID: profileID, Domain: domain}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(entry).Error
}

func (r *repository) RemoveProfileWhitelist(ctx context.Context, profileID uint, domain string) error {
	return r.db.WithContext(ctx).Where("profile_id = ? AND domain = ?", profileID, domain).Delete(&database.ProfileWhitelist{}).Error
}

func (r *repository) GetAllSubnets(ctx context.Context) ([]SubnetRule, error) {
	var rows []struct {
		ID          uint
		CIDR        string
		ProfileID   uint
		ProfileName string
	}
	err := r.db.WithContext(ctx).
		Table("subnet_profiles sp").
		Select("sp.id, sp.c_id_r as cidr, sp.profile_id, p.name as profile_name").
		Joins("JOIN profiles p ON p.id = sp.profile_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]SubnetRule, len(rows))
	for i, row := range rows {
		result[i] = SubnetRule{
			ID:          row.ID,
			CIDR:        row.CIDR,
			ProfileID:   row.ProfileID,
			ProfileName: row.ProfileName,
		}
	}
	return result, nil
}

func (r *repository) CreateSubnet(ctx context.Context, s *database.SubnetProfile) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *repository) UpdateSubnet(ctx context.Context, id uint, cidr string, profileID uint) error {
	return r.db.WithContext(ctx).Model(&database.SubnetProfile{}).Where("id = ?", id).Updates(map[string]any{
		"c_id_r":     cidr,
		"profile_id": profileID,
	}).Error
}

func (r *repository) DeleteSubnet(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&database.SubnetProfile{}, id).Error
}

func (r *repository) SetClientProfile(ctx context.Context, ip string, profileID *uint) error {
	result := r.db.WithContext(ctx).Model(&database.MacAddress{}).Where("ip = ?", ip).Update("profile_id", profileID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// No mac_addresses row exists for this IP (MAC not resolved via ARP).
		// Create a minimal stub row so the profile assignment can be stored.
		stub := &database.MacAddress{MAC: "ip:" + ip, IP: ip, ProfileID: profileID}
		return r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "mac"}},
			DoUpdates: clause.AssignmentColumns([]string{"profile_id", "ip"}),
		}).Create(stub).Error
	}
	return nil
}
